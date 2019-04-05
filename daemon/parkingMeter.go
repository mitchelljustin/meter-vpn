package daemon

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"time"
)

const TimeFormat = time.RFC1123

type ParkingMeter struct {
	store           PeerStore
	ctx             context.Context
	lnClient        lnrpc.LightningClient
	pendingInvoices map[string]pendingExtension
	priceTracker    PriceTracker
}

type LNDParams struct {
	Hostname     string
	MacaroonPath string
	CertPath     string
}

func NewParkingMeter(store PeerStore, lndParams LNDParams) (*ParkingMeter, error) {
	creds, err := credentials.NewClientTLSFromFile(lndParams.CertPath, "")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(lndParams.Hostname, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	macaroon, err := ioutil.ReadFile(lndParams.MacaroonPath)
	if err != nil {
		return nil, err
	}
	macaroonHex := hex.EncodeToString(macaroon)
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)
	booth := ParkingMeter{
		store:           store,
		priceTracker:    PriceTracker{},
		ctx:             ctx,
		lnClient:        lnrpc.NewLightningClient(conn),
		pendingInvoices: make(map[string]pendingExtension),
	}
	return &booth, nil
}

type pendingExtension struct {
	AccountID string
	Duration  time.Duration
	Completed chan bool
}

type ExtensionJSON struct {
	Duration string `json:"duration"`
}

type SetPubkeyJSON struct {
	PublicKey string `json:"publicKey"`
}

func respondBadRequest(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func respondServerError(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (pm *ParkingMeter) loadPeerFromCookie(ctx *gin.Context) (*Peer, bool) {
	accountId, err := ctx.Cookie("accountId")
	if err != nil {
		respondServerError(ctx, err)
		return nil, false
	}
	peer, err := pm.store.GetPeer(accountId)
	if err == ErrPeerNotFound {
		ctx.Status(404)
		return nil, false
	} else if err != nil {
		respondServerError(ctx, err)
		return nil, false
	}
	return peer, true
}

func (pm *ParkingMeter) Run() {
	sub, err := pm.lnClient.SubscribeInvoices(pm.ctx, &lnrpc.InvoiceSubscription{
		AddIndex:    0,
		SettleIndex: 0,
	})
	if err != nil {
		log.Fatalf("Could not subscribe to invoices: %v", err)
	}
	for {
		invoice, err := sub.Recv()
		if err != nil {
			log.Printf("Error receiving invoice: %v", err)
			continue
		}
		if invoice.State != lnrpc.Invoice_SETTLED {
			continue
		}
		payReq := invoice.PaymentRequest
		var extension pendingExtension
		var ok bool
		if extension, ok = pm.pendingInvoices[payReq]; !ok {
			continue
		}
		log.Printf("Adding %v of VPN time to %v", extension.Duration, extension.AccountID)
		peer, err := pm.store.GetPeer(extension.AccountID)
		if err != nil {
			continue
		}
		peer.AddAllowance(extension.Duration)
		if err = pm.store.SavePeer(peer); err != nil {
			log.Printf("Error saving peer: %v", err)
		}
		extension.Completed <- true
		delete(pm.pendingInvoices, payReq)
	}
}

func (pm *ParkingMeter) HandleExtensionRequest(ctx *gin.Context) {
	peer, ok := pm.loadPeerFromCookie(ctx)
	if !ok {
		return
	}
	var extension ExtensionJSON
	if err := ctx.BindJSON(&extension); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	duration, err := time.ParseDuration(fmt.Sprintf("%vs", extension.Duration))
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	sats := float64(duration) / float64(time.Hour) * pm.priceTracker.RetrieveSnapshot().Satoshi.Hour
	invoice := lnrpc.Invoice{
		Value: int64(math.Ceil(sats)),
		Memo:  fmt.Sprintf("Add %v to MeterVPN allowance", duration),
	}
	resp, err := pm.lnClient.AddInvoice(pm.ctx, &invoice)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	payReq := resp.PaymentRequest
	pm.pendingInvoices[payReq] = pendingExtension{
		AccountID: peer.AccountID,
		Duration:  duration,
		Completed: make(chan bool),
	}
	ctx.String(402, payReq)
}

func (pm *ParkingMeter) HandleExtensionCompletedRequest(ctx *gin.Context) {
	payReq := ctx.Query("payReq")
	var pending pendingExtension
	var ok bool
	if pending, ok = pm.pendingInvoices[payReq]; !ok {
		ctx.Status(404)
		return
	}
	timeout := time.NewTimer(time.Minute * 1)
	select {
	case <-pending.Completed:
		ctx.JSON(200, gin.H{
			"result": "completed",
		})
	case <-timeout.C:
		ctx.JSON(200, gin.H{
			"result": "timeout",
		})
	}
}

func (pm *ParkingMeter) HandleGetPeerRequest(ctx *gin.Context) {
	accountId, err := ctx.Cookie("accountId")
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	if accountId == "" {
		respondBadRequest(ctx, errors.New("missing accountId"))
		return
	}
	peer, err := pm.store.GetPeer(accountId)
	if err == ErrPeerNotFound {
		ctx.Status(404)
		return
	} else if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(200, PeerToJSON(peer))
}

func PeerToJSON(peer *Peer) gin.H {
	var publicKey, ipv4, ipv6 string
	if peer.PublicKeyB64 != nil {
		publicKey = *peer.PublicKeyB64
	}
	if peer.IPv4 != nil {
		ipv4 = peer.IPv4.String()
	}
	if peer.IPv6 != nil {
		ipv4 = peer.IPv6.String()
	}
	return gin.H{
		"accountId": peer.AccountID,
		"publicKey": publicKey,
		"ip": gin.H{
			"v4": ipv4,
			"v6": ipv6,
		},
		"expiryDate": peer.ExpiryDate.Format(TimeFormat),
	}
}

func (pm *ParkingMeter) HandleCreatePeerRequest(ctx *gin.Context) {
	peer, err := pm.store.CreatePeer()
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(200, PeerToJSON(peer))
}

func (pm *ParkingMeter) HandleSetPubkeyRequest(ctx *gin.Context) {
	peer, ok := pm.loadPeerFromCookie(ctx)
	if !ok {
		return
	}
	var config SetPubkeyJSON
	if err := ctx.ShouldBindJSON(&config); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	if _, err := KeyFromBase64(config.PublicKey); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	peer.PublicKeyB64 = &config.PublicKey
	if err := pm.store.SavePeer(peer); err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(200, nil)
}

func (pm *ParkingMeter) HandleIPRequest(ctx *gin.Context) {
	peer, ok := pm.loadPeerFromCookie(ctx)
	if !ok {
		return
	}
	if peer.IPv4 == nil {
		ips, err := pm.store.GetNewIPs()
		if err != nil {
			respondServerError(ctx, err)
			return
		}
		peer.IPv4 = &ips[0]
		peer.IPv6 = &ips[1]
		if err := pm.store.SavePeer(peer); err != nil {
			respondServerError(ctx, err)
			return
		}
	}
	ipv6Str := ""
	if peer.IPv6 != nil && *peer.IPv6 != nil {
		ipv6Str = peer.IPv6.String()
	}
	ctx.JSON(200, gin.H{
		"ipv4": peer.IPv4.String(),
		"ipv6": ipv6Str,
	})
}

func (pm *ParkingMeter) HandlePriceRequest(ctx *gin.Context) {
	ctx.JSON(200, pm.priceTracker.RetrieveSnapshot())
}
