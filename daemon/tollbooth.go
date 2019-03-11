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

const SatsPerMin = 5.8

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
	Pubkey   PublicKey
	Duration time.Duration
}

type Extension struct {
	Pubkey   string `json:"pubkey"`
	Duration string `json:"duration"`
}

type GetPeer struct {
	Pubkey string `json:"pubkey"`
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
		_, err = tb.store.AddAllowance(extension.Pubkey, extension.Duration)
		if err != nil {
			log.Printf("Error adding allowance: %v", err)
		}
		delete(tb.pendingInvoices, payReq)
	}
}

func (tb *TollBooth) HandleExtensionRequest(ctx *gin.Context) {
	var extension Extension
	if err := ctx.BindJSON(&extension); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	pubkey, err := UnmarshalPublicKey(extension.Pubkey)
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	duration, err := time.ParseDuration(fmt.Sprintf("%vs", extension.Duration))
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	sats := float64(duration) / float64(time.Minute) * SatsPerMin
	invoice := lnrpc.Invoice{
		Value: int64(math.Ceil(sats)),
		Memo:  fmt.Sprintf("meter-vpn: +%v", duration),
	}
	resp, err := tb.lnClient.AddInvoice(tb.ctx, &invoice)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	payReq := resp.PaymentRequest
	tb.pendingInvoices[payReq] = pendingExtension{
		Pubkey:   *pubkey,
		Duration: duration,
	}
	ctx.String(402, "lightning:%v", payReq)
}

func (tb *TollBooth) HandleGetPeerRequest(ctx *gin.Context) {
	var getPeer GetPeer
	if err := ctx.BindJSON(&getPeer); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	pubkey, err := UnmarshalPublicKey(getPeer.Pubkey)
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	expiry, err := tb.store.GetExpiry(*pubkey)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	if expiry == nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ip, err := tb.store.GetIPAddress(*pubkey)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"expiry": expiry.Format(TimeFormat),
		"ip":     ip.String(),
	})
}
