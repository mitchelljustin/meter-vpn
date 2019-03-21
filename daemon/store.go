package daemon

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/mdlayher/wireguardctrl/wgtypes"
	"math/big"
	"net"
	"time"
)

type Peer struct {
	gorm.Model

	AccountID  string `gorm:"type:varchar(64);unique_index"`
	ExpiryDate time.Time

	PublicKeyB64 *string `gorm:"unique_index"`
	IPv4         *net.IP
	IPv6         *net.IP
	Connected    bool
}

var ErrPeerNotFound = errors.New("peer not found")

const AccountIdSize = 10

type PeerStore interface {
	CreatePeer() (*Peer, error)
	GetPeer(accountId string) (*Peer, error)
	SavePeer(peer *Peer) error

	GetPeers(connected bool) ([]Peer, error)
	GetNewIPs() ([2]net.IP, error)
}

type SQLitePeerStore struct {
	DB *gorm.DB
}

func MigrateSQLModels(db *gorm.DB) {
	db.AutoMigrate(&Peer{})
}

func (store *SQLitePeerStore) CreatePeer() (*Peer, error) {
	var accountIdBytes [AccountIdSize]byte
	rand.Read(accountIdBytes[:])
	accountId := base32.StdEncoding.EncodeToString(accountIdBytes[:])
	newPeer := &Peer{
		AccountID:  accountId,
		ExpiryDate: time.Now(),
	}
	if err := store.DB.Save(newPeer).Error; err != nil {
		return nil, err
	}
	return newPeer, nil
}

func (store *SQLitePeerStore) GetPeer(accountId string) (*Peer, error) {
	var peer Peer
	if err := store.DB.First(&peer, &Peer{AccountID: accountId}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrPeerNotFound
		}
		return nil, err
	}
	return &peer, nil
}

func (store *SQLitePeerStore) SavePeer(peer *Peer) error {
	return store.DB.Save(peer).Error
}

func (store *SQLitePeerStore) GetPeers(connected bool) ([]Peer, error) {
	var peers []Peer
	if err := store.DB.
		Where("publicKeyB64 is not null").
		Where(Peer{Connected: connected}).
		Find(&peers).
		Error; err != nil {
		return nil, err
	}
	return peers, nil
}

func (p *Peer) AddAllowance(duration time.Duration) {
	p.ExpiryDate = p.ExpiryDate.Add(duration)
}

func KeyFromBase64(keyBase64 string) (*wgtypes.Key, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	key, err := wgtypes.NewKey(keyBytes)
	return &key, err
}

func ipToBigInt(ip net.IP) *big.Int {
	x := big.Int{}
	x.SetBytes([]byte(ip))
	return &x
}

func (store *SQLitePeerStore) GetNewIPs() (ips [2]net.IP, err error) {
	ips[0] = nil
	ips[1] = nil // TODO: ipv6
	var lastPeer Peer
	err = store.DB.
		Where("ipv4 is not null").
		Order("ipv4 asc").
		Last(&lastPeer).
		Error
	if err == gorm.ErrRecordNotFound {
		ips[0] = net.ParseIP("10.0.0.2").To4()
		err = nil
		return
	} else if err != nil {
		return
	}
	ipAsInt := ipToBigInt(*lastPeer.IPv4)
	ipAsInt.Add(ipAsInt, big.NewInt(1))
	ips[0] = ipAsInt.Bytes()
	if ipAsInt.Cmp(ipToBigInt(net.ParseIP("10.255.255.254"))) == 0 {
		err = errors.New("exhausted IPv4 space")
		return
	}
	return
}
