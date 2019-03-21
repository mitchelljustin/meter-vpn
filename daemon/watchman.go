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

	wireGuard *wireguardctrl.Client
}

func (w *Watchman) Report(format string, v ...interface{}) {
	log.Printf("[WATCHMAN] %v", fmt.Sprintf(format, v...))
}

func (w *Watchman) Run() {
	log.Printf("Running Watchman at interval: %v", w.Interval)
	w.ConnectToWireGuard()
	defer w.wireGuard.Close()

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

	w.wireGuard = client
}

func (w *Watchman) Tick() {
	now := time.Now()
	w.Report("TICK: %v", now)
	device, err := w.wireGuard.Device(WireguardDeviceName)
	if err != nil {
		w.Report("Error getting WireGuard device: %v", err)
		return
	}

	var disconnectedPeers, connectedPeers []Peer
	if connectedPeers, err = w.Store.GetPeers(true); err != nil {
		w.Report("Error getting connectedPeers: %v", err)
		return
	}
	if disconnectedPeers, err = w.Store.GetPeers(false); err != nil {
		w.Report("Error getting disconnectedPeers: %v", err)
		return
	}
	for _, peer := range connectedPeers {
		w.Report("Checking connected peer %v (expiry %v)", peer.AccountID, peer.ExpiryDate)
		if now.Before(peer.ExpiryDate) {
			continue
		}
		w.Report("Peer %v is out of allowance", peer.AccountID)
		err := w.DisconnectPeer(&peer)
		if err != nil {
			w.Report("ERROR: Could not disconnect peer, %v", err)
		}
		if err := w.Store.SavePeer(&peer); err != nil {
			w.Report("ERROR saving peer: %v", err)
		}
	}
	for _, peer := range disconnectedPeers {
		w.Report("Checking disconnected peer %v (expiry %v)", peer.AccountID, peer.ExpiryDate)
		if now.After(peer.ExpiryDate) {
			continue
		}
		key, err := KeyFromBase64(*peer.PublicKeyB64)
		if err != nil {
			w.Report("ERROR: %v", err)
			continue
		}
		found := false
		for _, devicePeer := range device.Peers {
			if devicePeer.PublicKey == *key {
				found = true
				break
			}
		}
		if !found {
			err := w.ConnectPeer(&peer)
			if err != nil {
				w.Report("ERROR: Could not connect devicePeer, %v", err)
			}
		}
	}
}

func (w *Watchman) ConnectPeer(peer *Peer) error {
	w.Report("Connecting peer: %v", peer.PublicKeyB64)
	key, err := KeyFromBase64(*peer.PublicKeyB64)
	if err != nil {
		return err
	}
	var allowedIPs []net.IPNet
	allowedIPs = append(allowedIPs, net.IPNet{
		IP:   *peer.IPv4,
		Mask: net.CIDRMask(32, 32),
	})
	if peer.IPv6 != nil {
		allowedIPs = append(allowedIPs, net.IPNet{
			IP:   *peer.IPv6,
			Mask: net.CIDRMask(128, 128),
		})
	}
	if err := w.wireGuard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  *key,
				AllowedIPs: allowedIPs,
			},
		},
	}); err != nil {
		return err
	}
	peer.Connected = true
	return nil
}

func (w *Watchman) DisconnectPeer(peer *Peer) error {
	w.Report("Disconnecting peer: %v", peer.PublicKeyB64)
	key, err := KeyFromBase64(*peer.PublicKeyB64)
	if err != nil {
		return err
	}
	if err := w.wireGuard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: *key,
				Remove:    true,
			},
		},
	}); err != nil {
		return err
	}
	peer.Connected = false
	return nil
}
