package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/mvanderh/meter-vpn/daemon"
	"log"
	"time"
)

func main() {
	port := flag.Int("p", 8000, "port")
	dbPath := flag.String("f", "data/meter.db", "database path")
	watchInterval := flag.Uint("i", 15, "watch interval in seconds")
	debugMode := flag.Bool("d", false, "debug mode")
	flag.Parse()

	if *debugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := gorm.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	defer db.Close()
	daemon.MigrateSQLModels(db)

	store := &daemon.SQLitePeerStore{DB: db}

	booth, err := daemon.NewTollBooth(store, daemon.LNDParams{
		MacaroonPath: "secret/admin.macaroon",
		CertPath:     "secret/tls.cert",
		Hostname:     "159.89.121.214:10009",
	})
	if err != nil {
		log.Fatal(err)
	}
	go booth.Run()

	watchman := daemon.Watchman{
		Store:    store,
		Interval: time.Duration(*watchInterval) * time.Second,
	}
	go watchman.Run()

	startGinServer(booth, *port)
}

func startGinServer(booth *daemon.TollBooth, port int) {
	router := gin.Default()

	router.POST("/peer", booth.HandleCreatePeerRequest)
	router.GET("/peer", booth.HandleGetPeerRequest)
	router.GET("/peer/ip", booth.HandleIPRequest)
	router.POST("/peer/pubkey", booth.HandleSetPubkeyRequest)
	router.POST("/peer/extend", booth.HandleExtensionRequest)

	router.Use(static.ServeRoot("/", "./www"))
	router.GET("/account", func(ctx *gin.Context) {
		ctx.File("./www/account.html")
	})

	addr := fmt.Sprintf(":%v", port)
	log.Printf("Server running at %v", addr)
	log.Fatal(router.Run(addr))
}
