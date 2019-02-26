package metervpn

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const TimeFormat = time.RFC1123

type TollBooth struct {
	Store AllowanceStore
}

type Extension struct {
	Pubkey   string `json:"pubkey"`
	Duration string `json:"duration"`
}

type GetExpiry struct {
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

func (tb *TollBooth) HandleGetExpiryRequest(ctx *gin.Context) {
	var getExpiry GetExpiry
	if err := ctx.BindJSON(&getExpiry); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	pubkey, err := UnmarshalPublicKey(getExpiry.Pubkey)
	if err != nil {
		respondBadRequest(ctx, err)
		return
	}
	expiry, err := tb.Store.GetExpiry(*pubkey)
	if err != nil {
		respondServerError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"expiry": expiry.Format(TimeFormat)})
}
