package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
	"net"
	"time"
)

type PeerStore interface {
	AddAllowance(pubkey PublicKey, duration time.Duration) (*time.Time, error)
	GetExpiry(pubkey PublicKey) (*time.Time, error)
	GetIPAddress(pubkey PublicKey) (*net.IP, error)

	GetAllPubkeys() ([]PublicKey, error)
}

type LevelDBPeerRecord struct {
	Expiry string `json:"expiry"`
	IPv6   string `json:"ipv6"`
}

type LevelDBPeerStore struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC3339

const IPv6Prefix = "fddf:cbfb:7aa3:0001"

const HighestIPKey = "/highestIP"

func keyForPubkey(pubkey PublicKey) []byte {
	key := fmt.Sprintf("/peer/%v", MarshalPublicKey(pubkey))
	return []byte(key)
}

func (s *LevelDBPeerStore) savePeer(pubkey PublicKey, peer *LevelDBPeerRecord) error {
	obj, err := json.Marshal(peer)
	if err != nil {
		return err
	}
	key := keyForPubkey(pubkey)
	return s.DB.Put(key, obj, nil)
}

func (s *LevelDBPeerStore) getOrCreatePeer(pubkey PublicKey) (*LevelDBPeerRecord, error) {
	dbKey := keyForPubkey(pubkey)
	bytes, err := s.DB.Get(dbKey, nil)
	if err == leveldb.ErrNotFound {
		record := LevelDBPeerRecord{}
		return &record, nil
	} else if err != nil {
		return nil, err
	}
	var record LevelDBPeerRecord
	if err := json.Unmarshal(bytes, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *LevelDBPeerStore) AddAllowance(pubkey PublicKey, duration time.Duration) (*time.Time, error) {
	expiry, err := s.GetExpiry(pubkey)
	if err != nil {
		return nil, err
	}
	if expiry == nil {
		now := time.Now()
		expiry = &now
	}
	newExpiry := expiry.Add(duration)
	peer, err := s.getOrCreatePeer(pubkey)
	if err != nil {
		return nil, err
	}
	peer.Expiry = newExpiry.Format(TimeLayout)
	if err := s.savePeer(pubkey, peer); err != nil {
		return nil, err
	}
	return &newExpiry, nil
}

func (s *LevelDBPeerStore) GetExpiry(pubkey PublicKey) (*time.Time, error) {
	peer, err := s.getOrCreatePeer(pubkey)
	if err != nil {
		return nil, err
	}
	if err != nil || peer.Expiry == "" {
		return nil, err
	}
	expiry, err := time.Parse(TimeLayout, peer.Expiry)
	if err != nil {
		return nil, err
	}
	return &expiry, nil
}

func (s *LevelDBPeerStore) GetAllPubkeys() ([]PublicKey, error) {
	iter := s.DB.NewIterator(util.BytesPrefix([]byte("/peer/")), nil)
	var pubkeys []PublicKey = nil
	for iter.Next() {
		pubkeyStr := string(iter.Key())[6:]
		pubkey, err := UnmarshalPublicKey(pubkeyStr)
		if err != nil {
			return nil, err
		}
		pubkeys = append(pubkeys, *pubkey)
	}
	return pubkeys, nil
}

func (s *LevelDBPeerStore) GetIPAddress(pubkey PublicKey) (*net.IP, error) {
	peer, err := s.getOrCreatePeer(pubkey)
	if err != nil {
		return nil, err
	}
	if peer.IPv6 == "" {
		highestIP, err := s.DB.Get([]byte(HighestIPKey), nil)
		if err == leveldb.ErrNotFound {
			highestIP = net.ParseIP(fmt.Sprintf("%v::1", IPv6Prefix))
		} else if err != nil {
			return nil, err
		}
		ipInt := new(big.Int).SetBytes(highestIP)
		ipInt.Add(ipInt, big.NewInt(1))
		newIP := net.IP(ipInt.Bytes())
		if newIP.Equal(net.ParseIP(fmt.Sprintf("%v:ffff:ffff:ffff:ffff", IPv6Prefix))) {
			// Overflow
			return nil, errors.New("out of ipv6 addresses")
		}
		if err := s.DB.Put([]byte(HighestIPKey), newIP[:], nil); err != nil {
			return nil, err
		}
		peer.IPv6 = newIP.String()
		if err := s.savePeer(pubkey, peer); err != nil {
			return nil, err
		}
	}
	peerIP := net.ParseIP(peer.IPv6)
	return &peerIP, nil
}
