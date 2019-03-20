package daemon

import (
	"encoding/hex"
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

const SatsPerHour = 250

type TollBooth struct {
	store           PeerStore
	ctx             context.Context
	lnClient        lnrpc.LightningClient
	pendingInvoices map[string]pendingExtension
}

type LNDParams struct {
	Hostname     string
	MacaroonPath string
	CertPath     string
}

func NewTollBooth(store PeerStore, lndParams LNDParams) (*TollBooth, error) {
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
	booth := TollBooth{
		store:           store,
		ctx:             ctx,
		lnClient:        lnrpc.NewLightningClient(conn),
		pendingInvoices: make(map[string]pendingExtension),
	}
	return &booth, nil
}

type pendingExtension struct {
	AccountID string
	Duration  time.Duration
}

type ExtensionJSON struct {
	Duration string `json:"duration"`
}

type VPNConfigJSON struct {
	PublicKey string `json:"publicKey"`
	IP        struct {
		V4 string
		V6 string
	}
}

func respondBadRequest(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func respondServerError(ctx *gin.Context, err error) {
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (tb *TollBooth) Run() {
	sub, err := tb.lnClient.SubscribeInvoices(tb.ctx, &lnrpc.InvoiceSubscription{
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
		if extension, ok = tb.pendingInvoices[payReq]; !ok {
			continue
		}
		peer, err := tb.store.GetPeer(extension.AccountID)
		if err != nil {
			continue
		}
		peer.AddAllowance(extension.Duration)
		if err = tb.store.SavePeer(peer); err != nil {
			log.Printf("Error saving peer: %v", err)
		}
		delete(tb.pendingInvoices, payReq)
	}
}

func (tb *TollBooth) HandleExtensionRequest(ctx *gin.Context) {
	var extension ExtensionJSON
	if err := ctx.BindJSON(&extension); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	accountId := ctx.Param("accountId")
	peer, err := tb.store.GetPeer(accountId)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	if peer == nil {
		ctx.Status(404)
		return
	}
	duration, err := time.ParseDuration(fmt.Sprintf("%vs", extension.Duration))
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	sats := float64(duration) / float64(time.Hour) * SatsPerHour
	invoice := lnrpc.Invoice{
		Value: int64(math.Ceil(sats)),
		Memo:  fmt.Sprintf("Add %v to MeterVPN allowance", duration),
	}
	resp, err := tb.lnClient.AddInvoice(tb.ctx, &invoice)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	payReq := resp.PaymentRequest
	tb.pendingInvoices[payReq] = pendingExtension{
		AccountID: accountId,
		Duration:  duration,
	}
	ctx.String(402, payReq)
}

func (tb *TollBooth) HandleGetPeerRequest(ctx *gin.Context) {
	accountId := ctx.Param("accountId")
	peer, err := tb.store.GetPeer(accountId)
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

func (tb *TollBooth) HandleCreatePeerRequest(ctx *gin.Context) {
	peer, err := tb.store.CreatePeer()
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(200, PeerToJSON(peer))
}

func (tb *TollBooth) HandleSetConfigRequest(ctx *gin.Context) {
	var config VPNConfigJSON
	if err := ctx.ShouldBindJSON(&config); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	// TODO
	ctx.Status(200)
}
