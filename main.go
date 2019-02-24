package main

import (
	"MeterVPN/lib"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"net/http"
	"os"
	"time"
)

type Extend struct {
	Pubkey   string `json:"pubkey"`
	Duration string `json:"duration"`
}

func main() {
	port := flag.Int("p", 8000, "port")
	dbPath := flag.String("d", "data/meter.db", "database path")
	flag.Parse()

	router := gin.Default()

	router.Static("/", "./www")

	db, err := leveldb.OpenFile(*dbPath, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer db.Close()

	store := lib.Store{DB: db}
	router.POST("/extend", func(ctx *gin.Context) {
		var extend Extend
		if err := ctx.BindJSON(&extend); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		pubkeyBytes, err := base64.StdEncoding.DecodeString(extend.Pubkey)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if len(pubkeyBytes) != 32 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "pubkeyBytes must be 32 bytes"})
			return
		}
		var pubkey lib.PublicKey
		copy(pubkey[:lib.PublicKeySize], pubkeyBytes)
		duration, err := time.ParseDuration(fmt.Sprintf("%vs", extend.Duration))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		expiry, err := store.AddDuration(pubkey, duration)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"newExpiry": expiry.Format(time.RFC1123)})
	})

	log.Fatal(router.Run(fmt.Sprintf(":%v", *port)))
}
