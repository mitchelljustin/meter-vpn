package lib

import (
	"fmt"
	"github.com/mdlayher/wireguardctrl"
	"log"
	"time"
)

type Watchman struct {
	Store ExpiryStore

	wgClient *wireguardctrl.Client
}

func (w *Watchman) Report(format string, v ...interface{}) {
	log.Printf("[WATCHMAN] %v", fmt.Sprintf(format, v))
}

func RunWatchman(interval time.Duration, store ExpiryStore) {
	log.Printf("Running Watchman at interval: %v", interval)
	watchman := Watchman{
		Store: store,
	}
	watchman.ConnectToWireGuard()
	defer watchman.wgClient.Close()

	ticker := time.NewTicker(interval)
	for {
		watchman.Tick()
		<-ticker.C
	}
}

func (w *Watchman) ConnectToWireGuard() {
	client, err := wireguardctrl.New()

	if err != nil {
		w.Report("Could not connect to Wireguard: %v", err)
		return
	}

	w.wgClient = client
}

func (w *Watchman) Tick() {
	w.Report("Checking at %v", time.Now())

	dev, err := w.wgClient.Device("wg0")
	if err != nil {
		w.Report("Error getting WireGuard device: %v", err)
		return
	}
	w.Report("Device: %v", dev)
}
