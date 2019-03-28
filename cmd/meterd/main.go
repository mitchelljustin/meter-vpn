package main

import (
	"encoding/json"
	"flag"
	"fmt"
	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/mvanderh/meter-vpn/daemon"
	"log"
	"net/http"
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

	createApiRoutes(router, booth)

	createWwwRoutes(router)

	addr := fmt.Sprintf(":%v", port)
	log.Printf("Server running at %v", addr)
	log.Fatal(router.Run(addr))
}

const (
	MonthlyCostUSD = 4.00
	DailyCostUSD   = MonthlyCostUSD / 30.5
	HourlyCostUSD  = DailyCostUSD / 24
)

type coindeskCurrentPrice struct {
	Bpi struct {
		USD struct {
			RateFloat float64 `json:"rate_float"`
		} `json:"USD"`
	}
}

func costToSatoshi(costInUsd, rate float64) string {
	return fmt.Sprintf("%.3f", costInUsd/rate*1e8)
}

func costToUsd(costInUsd float64) string {
	return fmt.Sprintf("%.5f", costInUsd)
}

func priceHandler() func(ctx *gin.Context) {
	lastBtcPrice := 3936.6467

	return func(ctx *gin.Context) {
		resp, err := http.Get("https://api.coindesk.com/v1/bpi/currentprice/USD.json")
		rate := lastBtcPrice
		if err == nil {
			defer resp.Body.Close()
			var curPrice coindeskCurrentPrice
			if err := json.NewDecoder(resp.Body).Decode(&curPrice); err == nil {
				rate = curPrice.Bpi.USD.RateFloat
				lastBtcPrice = rate
			} else {
				log.Printf("Error decoding JSON: %v", err)
			}
		} else {
			log.Printf("Error getting price: %v", err)
		}
		ctx.JSON(200, gin.H{
			"satoshi": gin.H{
				"month": costToSatoshi(MonthlyCostUSD, rate),
				"day":   costToSatoshi(DailyCostUSD, rate),
				"hour":  costToSatoshi(HourlyCostUSD, rate),
			},
			"usd": gin.H{
				"month": costToUsd(MonthlyCostUSD),
				"day":   costToUsd(DailyCostUSD),
				"hour":  costToUsd(HourlyCostUSD),
			},
		})
	}
}

func createApiRoutes(router *gin.Engine, booth *daemon.TollBooth) {
	router.GET("/price", priceHandler())
	router.POST("/peer", booth.HandleCreatePeerRequest)
	router.GET("/peer", booth.HandleGetPeerRequest)
	router.GET("/peer/ip", booth.HandleIPRequest)
	router.POST("/peer/pubkey", booth.HandleSetPubkeyRequest)
	router.POST("/peer/extend", booth.HandleExtensionRequest)
}

type pageInfo struct {
	Title string
	File  string
}

var Pages = map[string]pageInfo{
	"": {
		File:  "index",
		Title: "MeterVPN - Anonymous, pro-rated VPN",
	},
	"account": {
		File:  "account",
		Title: "MeterVPN - My Account",
	},
	"login": {
		File:  "login",
		Title: "MeterVPN - Log In",
	},
	"create-account": {
		File:  "create-account",
		Title: "MeterVPN - Create account",
	},
	"faq": {
		File:  "faq",
		Title: "MeterVPN - Frequently Asked Questions",
	},
	"about": {
		File:  "about",
		Title: "MeterVPN - About",
	},
}

func createWwwRoutes(router *gin.Engine) {
	router.HTMLRender = gintemplate.New(gintemplate.TemplateConfig{
		Root:         "views",
		Extension:    ".html",
		Master:       "layouts/master",
		DisableCache: true,
	})
	for name, info := range Pages {
		path := "/" + name
		func(path string, info pageInfo) {
			router.GET(path, func(ctx *gin.Context) {
				accountId, err := ctx.Cookie("accountId")
				loggedIn := err != http.ErrNoCookie && accountId != ""
				ctx.HTML(http.StatusOK, info.File, gin.H{
					"Title":    info.Title,
					"LoggedIn": loggedIn,
				})
			})
		}(path, info)
	}
	router.Use(static.ServeRoot("/", "./www"))
}
