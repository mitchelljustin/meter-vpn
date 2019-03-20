package daemon

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/mdlayher/wireguardctrl/wgtypes"
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

	GetConnectedPeers() ([]Peer, error)
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

func (store *SQLitePeerStore) GetConnectedPeers() ([]Peer, error) {
	var peers []Peer
	if err := store.DB.Find(&peers, Peer{Connected: true}).Error; err != nil {
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
