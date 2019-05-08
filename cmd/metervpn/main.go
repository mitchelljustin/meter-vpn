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
	"os"
	"time"
)

func main() {
	port := flag.Int("p", 8000, "port")
	dbPath := flag.String("f", "data/meter.db", "database path")
	watchInterval := flag.Uint("i", 15, "watch interval in seconds")
	debugMode := flag.Bool("d", false, "debug mode")
	logPath := flag.String("l", "-", "log path")
	flag.Parse()

	if *debugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if *logPath != "-" {
		logFile, err := os.OpenFile(*logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	db, err := gorm.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	defer db.Close()
	daemon.MigrateSQLModels(db)

	store := &daemon.SQLitePeerStore{DB: db}

	vpnMeter, err := daemon.NewVPNMeter(store, daemon.LNDParams{
		MacaroonPath: "secret/admin.macaroon",
		CertPath:     "secret/tls.cert",
		Hostname:     "localhost:10009",
	})
	if err != nil {
		log.Fatal(err)
	}
	go vpnMeter.Run()

	wgAgent := daemon.WireGuardAgent{
		Store:    store,
		Interval: time.Duration(*watchInterval) * time.Second,
	}
	go wgAgent.Run()

	app := createHTTPServer(vpnMeter)

	addr := fmt.Sprintf(":%v", port)
	log.Printf("Server running at %v", addr)

	log.Fatal(app.Run(addr))
}

func createHTTPServer(meter *daemon.VPNMeter) *gin.Engine {
	app := gin.Default()

	if gin.Mode() == gin.ReleaseMode {
		rate, _ := limiter.NewRateFromFormatted("1000-H")
		lim := limiter.New(memory.NewStore(), rate)
		app.Use(mgin.NewMiddleware(lim))
	}

	addAPIRoutes(app, meter)
	addWWWRoutes(app)

	return app
}

func addAPIRoutes(router *gin.Engine, meter *daemon.VPNMeter) {
	router.GET("/price", meter.HandlePriceRequest)
	router.POST("/peer", meter.HandleCreatePeerRequest)
	router.GET("/peer", meter.HandleGetPeerRequest)
	router.GET("/peer/ip", meter.HandleIPRequest)
	router.POST("/peer/pubkey", meter.HandleSetPubkeyRequest)
	router.POST("/peer/extend", meter.HandleExtensionRequest)
	router.GET("/peer/extend/completed", meter.HandleExtensionCompletedRequest)
}

type pageInfo struct {
	Path  string
	Title string
	File  string
}

var Pages = []pageInfo{
	{
		Path:  "/",
		File:  "index",
		Title: "MeterVPN - Pay-as-you-go VPN",
	},
	{
		Path:  "/account",
		File:  "account",
		Title: "MeterVPN - My Account",
	},
	{
		Path:  "/account/welcome",
		File:  "account-welcome",
		Title: "MeterVPN - Welcome!",
	},
}

func addWWWRoutes(router *gin.Engine) {
	disableCache := gin.Mode() != gin.ReleaseMode
	router.HTMLRender = gintemplate.New(gintemplate.TemplateConfig{
		Root:         "views",
		Extension:    ".html",
		Master:       "layouts/master",
		DisableCache: disableCache,
	})
	for _, page := range Pages {
		func(page pageInfo) {
			router.GET(page.Path, func(ctx *gin.Context) {
				accountId, err := ctx.Cookie("accountId")
				loggedIn := err != http.ErrNoCookie && accountId != ""
				ctx.HTML(http.StatusOK, page.File, gin.H{
					"Title":    page.Title,
					"LoggedIn": loggedIn,
				})
			})
		}(page)
	}
	router.Use(static.ServeRoot("/", "./www"))
}
