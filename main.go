package main

import (
	"MeterVPN/metervpn"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"time"
)

func main() {
	port := flag.Int("p", 8080, "port")
	dbPath := flag.String("d", "data/meter.db", "database path")
	watchInterval := flag.Uint("i", 15, "watch interval in seconds")
	flag.Parse()

	db, err := leveldb.OpenFile(*dbPath, nil)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	defer db.Close()
	store := metervpn.LevelDBPeerStore{DB: db}

	booth, err := metervpn.NewTollBooth(&store, metervpn.LNDParams{
		MacaroonPath: "secret/admin.macaroon",
		CertPath:     "secret/tls.cert",
		Hostname:     "159.89.121.214:10009",
	})
	if err != nil {
		log.Fatal(err)
	}
	go booth.Run()

	watchman := metervpn.Watchman{
		Store:    &store,
		Interval: time.Duration(*watchInterval) * time.Second,
	}
	go watchman.Run()

	startGinServer(booth, *port)
}

func startGinServer(booth *metervpn.TollBooth, port int) {
	router := gin.Default()

	router.POST("/api/extend", booth.HandleExtensionRequest)
	router.GET("/api/peer", booth.HandleGetPeerRequest)

	router.Static("/app", "./www")

	addr := fmt.Sprintf(":%v", port)
	log.Fatal(router.Run(addr))
}
