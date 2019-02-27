package metervpn

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const TimeFormat = time.RFC1123

type TollBooth struct {
	Store PeerStore
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
	expiry, err := tb.Store.AddAllowance(*pubkey, duration)
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
	expiry, err := tb.Store.GetExpiry(*pubkey)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ip, err := tb.Store.GetIPAddress(*pubkey)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"expiry": expiry.Format(TimeFormat),
		"ip":     ip.String(),
	})
}
