package main

import (
	"flag"
	"fmt"
	gintemplate "github.com/foolin/gin-template"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/mvanderh/meter-vpn/daemon"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
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

	parkingMeter, err := daemon.NewParkingMeter(store, daemon.LNDParams{
		MacaroonPath: "secret/admin.macaroon",
		CertPath:     "secret/tls.cert",
		Hostname:     "159.89.121.214:10009",
	})
	if err != nil {
		log.Fatal(err)
	}
	go parkingMeter.Run()

	vpnAgent := daemon.VPNAgent{
		Store:    store,
		Interval: time.Duration(*watchInterval) * time.Second,
	}
	go vpnAgent.Run()

	startGinServer(parkingMeter, *port)
}

func startGinServer(booth *daemon.ParkingMeter, port int) {
	router := gin.Default()

	if gin.Mode() == gin.ReleaseMode {
		rate, _ := limiter.NewRateFromFormatted("1000-H")
		lim := limiter.New(memory.NewStore(), rate)
		router.Use(mgin.NewMiddleware(lim))
	}

	createApiRoutes(router, booth)

	createWwwRoutes(router)

	addr := fmt.Sprintf(":%v", port)
	log.Printf("Server running at %v", addr)
	log.Fatal(router.Run(addr))
}

func createApiRoutes(router *gin.Engine, meter *daemon.ParkingMeter) {
	router.GET("/price", meter.HandlePriceRequest)
	router.POST("/peer", meter.HandleCreatePeerRequest)
	router.GET("/peer", meter.HandleGetPeerRequest)
	router.GET("/peer/ip", meter.HandleIPRequest)
	router.POST("/peer/pubkey", meter.HandleSetPubkeyRequest)
	router.POST("/peer/extend", meter.HandleExtensionRequest)
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
