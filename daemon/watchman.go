package daemon

import (
	"fmt"
	"github.com/mdlayher/wireguardctrl"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"log"
	"net"
	"time"
)

const WireguardDeviceName = "wg0"

type Watchman struct {
	Store    PeerStore
	Interval time.Duration

	wireguard *wireguardctrl.Client
}

func (w *Watchman) Report(format string, v ...interface{}) {
	log.Printf("[WATCHMAN] %v", fmt.Sprintf(format, v...))
}

func (w *Watchman) Run() {
	log.Printf("Running Watchman at interval: %v", w.Interval)
	w.ConnectToWireGuard()
	defer w.wireguard.Close()

	ticker := time.NewTicker(w.Interval)
	for {
		w.Tick()
		<-ticker.C
	}
}

func (w *Watchman) ConnectToWireGuard() {
	client, err := wireguardctrl.New()

	if err != nil {
		w.Report("Could not connect to Wireguard: %v", err)
		return
	}

	w.wireguard = client
}

func (w *Watchman) Tick() {
	now := time.Now()
	w.Report("TICK %v", now)
	device, err := w.wireguard.Device(WireguardDeviceName)
	if err != nil {
		w.Report("Error getting WireGuard device: %v", err)
	}

	pubkeys, err := w.Store.GetAllPubkeys()
	if err != nil {
		w.Report("Error getting all pubkeys: %v", err)
		return
	}
	for _, pubkey := range pubkeys {
		pubkeyStr := MarshalPublicKey(pubkey)

		expiry, err := w.Store.GetExpiry(pubkey)
		if err != nil {
			w.Report("Error getting expiry for %v, %v", pubkeyStr, err)
			continue
		}
		if expiry == nil {
			continue
		}
		w.Report("Checking %v (expiry %v)", pubkeyStr, *expiry)
		if now.After(*expiry) {
			w.Report("Peer %v is out of allowance", pubkeyStr)
			err := w.DisconnectPeer(pubkey)
			if err != nil {
				w.Report("ERROR: Could not disconnect peer, %v", err)
			}
		} else {
			if device == nil {
				w.Report("Device not found, skipping peer connection")
				continue
			}
			found := false
			for _, peer := range device.Peers {
				if peer.PublicKey.String() == pubkeyStr {
					found = true
					break
				}
			}
			if !found {
				err := w.ConnectPeer(pubkey, device.Peers)
				if err != nil {
					w.Report("ERROR: Could not connect peer, %v", err)
				}
			}
		}
	}
}

func (w *Watchman) ConnectPeer(pubkey PublicKey, peers []wgtypes.Peer) error {
	w.Report("Connecting peer: %v", MarshalPublicKey(pubkey))
	ip, err := w.Store.GetIPAddress(pubkey)
	if err != nil {
		return err
	}
	ipNet := net.IPNet{
		IP:   *ip,
		Mask: net.CIDRMask(32, 32),
	}
	return w.wireguard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  pubkey,
				AllowedIPs: []net.IPNet{ipNet},
			},
		},
	})
}

func (w *Watchman) DisconnectPeer(pubkey PublicKey) error {
	w.Report("Disconnecting peer: %v", MarshalPublicKey(pubkey))
	err := w.wireguard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: pubkey,
				Remove:    true,
			},
		},
	})
	if err == nil {
		err = w.Store.Expire(pubkey)
	}
	return err
}