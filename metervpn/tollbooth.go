package metervpn

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
	"net/http"
	"time"
)

const TimeFormat = time.RFC1123

type TollBooth struct {
	store    PeerStore
	lnClient lnrpc.LightningClient
	ctx      context.Context
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
	lnClient := lnrpc.NewLightningClient(conn)
	booth := TollBooth{
		store:    store,
		lnClient: lnClient,
		ctx:      ctx,
	}
	booth.Test()
	return &booth, nil
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
	// TODO: generate lightning invoice
	expiry, err := tb.store.AddAllowance(*pubkey, duration)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"newExpiry": expiry.Format(TimeFormat)})
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

func (tb *TollBooth) Test() {
	resp, err := tb.lnClient.WalletBalance(tb.ctx, &lnrpc.WalletBalanceRequest{}, nil)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println(resp)
	}
}
