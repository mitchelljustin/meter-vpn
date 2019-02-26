package main

import (
	"MeterVPN/lib"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"os"
)

func main() {
	port := flag.Int("p", 8000, "port")
	dbPath := flag.String("d", "data/meter.db", "database path")
	flag.Parse()

	db, err := leveldb.OpenFile(*dbPath, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer db.Close()

	store := lib.LevelDBExpiryStore{DB: db}
	booth := lib.TollBooth{Store: &store}

	startGinServer(&booth, *port)
}

func startGinServer(booth *lib.TollBooth, port int) {
	router := gin.Default()

	router.POST("/api/extend", booth.HandleExtensionRequest)
	router.GET("/api/expiry", booth.HandleGetExpiryRequest)

	router.Static("/app", "./www")

	addr := fmt.Sprintf(":%v", port)
	log.Fatal(router.Run(addr))
}
