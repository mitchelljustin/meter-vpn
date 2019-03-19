package daemon

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"net"
	"time"
)

type Peer struct {
	gorm.Model
	AccountID  string `gorm:"type:varchar(64);unique_index"`
	PublicKey  PublicKey
	IPv4       net.IP
	IPv6       net.IP
	ExpiryDate time.Time
}

type PeerStore struct {
	db *gorm.DB
}

func MigrateModels(db *gorm.DB) {
	db.AutoMigrate(&Peer{})
}

func (p *PeerStore) FindPeer(accountId string) (*Peer, error) {
	var peer Peer
	err := p.db.Find(&peer, &Peer{AccountID: accountId}).Error
	if err != nil {
		return nil, err
	}
	return &peer, nil
}

func (p *PeerStore) CreateNewPeer() Peer {

}
