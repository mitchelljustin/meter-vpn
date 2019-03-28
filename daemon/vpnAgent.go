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

type VPNAgent struct {
	Store    PeerStore
	Interval time.Duration

	wireGuard *wireguardctrl.Client
}

func (a *VPNAgent) Report(format string, v ...interface{}) {
	log.Printf("[WATCHMAN] %v", fmt.Sprintf(format, v...))
}

func (a *VPNAgent) Run() {
	log.Printf("Running VPNAgent at interval: %v", a.Interval)
	a.ConnectToWireGuard()
	defer a.wireGuard.Close()

	ticker := time.NewTicker(a.Interval)
	for {
		a.Tick()
		<-ticker.C
	}
}

func (a *VPNAgent) ConnectToWireGuard() {
	client, err := wireguardctrl.New()

	if err != nil {
		a.Report("Could not connect to Wireguard: %v", err)
		return
	}

	a.wireGuard = client
}

func (a *VPNAgent) Tick() {
	now := time.Now()
	a.Report("TICK: %v", now)
	device, err := a.wireGuard.Device(WireguardDeviceName)
	if err != nil {
		a.Report("Error getting WireGuard device: %v", err)
		return
	}

	connectedToWg := make(map[wgtypes.Key]bool)
	for _, devicePeer := range device.Peers {
		connectedToWg[devicePeer.PublicKey] = true
	}

	var peers []Peer
	if peers, err = a.Store.GetPeersWithKey(); err != nil {
		a.Report("Error getting peers: %v", err)
		return
	}
	for _, peer := range peers {
		a.Report("Checking peer %v (expiry %v)", peer.AccountID, peer.ExpiryDate)
		key, err := KeyFromBase64(*peer.PublicKeyB64)
		if err != nil {
			a.Report("ERROR: %v", err)
		}
		if now.Before(peer.ExpiryDate) && !connectedToWg[*key] {
			if err := a.ConnectPeer(&peer); err != nil {
				a.Report("Could not connect peer: %v", err)
			}
		}
		if now.After(peer.ExpiryDate) && connectedToWg[*key] {
			if err := a.DisconnectPeer(&peer); err != nil {
				a.Report("Could not disconnect peer: %v", err)
			}
		}
	}
}

func (a *VPNAgent) ConnectPeer(peer *Peer) error {
	a.Report("Connecting peer: %v", peer.AccountID)
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
	if err := a.wireGuard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  *key,
				AllowedIPs: allowedIPs,
			},
		},
	}); err != nil {
		return err
	}
	return nil
}

func (a *VPNAgent) DisconnectPeer(peer *Peer) error {
	a.Report("Disconnecting peer: %v", peer.AccountID)
	key, err := KeyFromBase64(*peer.PublicKeyB64)
	if err != nil {
		return err
	}
	if err := a.wireGuard.ConfigureDevice(WireguardDeviceName, wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: *key,
				Remove:    true,
			},
		},
	}); err != nil {
		return err
	}
	return nil
}
