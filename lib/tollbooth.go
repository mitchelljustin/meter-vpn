package lib

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

const TimeFormat = time.RFC1123

type TollBooth struct {
	Store ExpiryStore
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

func extractPubkey(base64Pubkey string) (*PublicKey, error) {
	pubkeyBytes, err := base64.StdEncoding.DecodeString(base64Pubkey)
	if err != nil {
		return nil, err
	}
	if len(pubkeyBytes) != 32 {
		return nil, errors.New("public key must be 32 bytes")
	}
	var pubkey PublicKey
	copy(pubkey[:PublicKeySize], pubkeyBytes)
	return &pubkey, nil
}

func (tb *TollBooth) HandleExtensionRequest(ctx *gin.Context) {
	var extension Extension
	if err := ctx.BindJSON(&extension); err != nil {
		respondBadRequest(ctx, err)
		return
	}
	pubkey, err := extractPubkey(extension.Pubkey)
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
	expiry, err := tb.Store.AddDuration(*pubkey, duration)
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
	pubkey, err := extractPubkey(getExpiry.Pubkey)
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
