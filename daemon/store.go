package daemon

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"time"
)

type PeerStore interface {
	AddAllowance(pubkey PublicKey, duration time.Duration) (*time.Time, error)
	GetExpiry(pubkey PublicKey) (*time.Time, error)
	GetIPAddress(pubkey PublicKey) (*net.IP, error)

	GetAllPubkeys() ([]PublicKey, error)
	Expire(pubkey PublicKey) error
}

type LevelDBPeerRecord struct {
	Expiry string `json:"expiry"`
	IP     string `json:"ip"`
}

type LevelDBPeerStore struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC3339

const HighestIPKey = "highestIP"

func keyForPubkey(pubkey PublicKey) []byte {
	return []byte(
		fmt.Sprintf("pubkey:%v", MarshalPublicKey(pubkey)),
	)
}

func (s *LevelDBPeerStore) savePeer(pubkey PublicKey, peer *LevelDBPeerRecord) error {
	obj, err := json.Marshal(peer)
	if err != nil {
		return err
	}
	return s.DB.Put(keyForPubkey(pubkey), obj, nil)
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
	if peer.Expiry == "" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	expiry, err := time.Parse(TimeLayout, peer.Expiry)
	if err != nil {
		return nil, err
	}
	return &expiry, nil
}

func (s *LevelDBPeerStore) Expire(pubkey PublicKey) error {
	peer, err := s.getOrCreatePeer(pubkey)
	if err != nil {
		return err
	}
	peer.Expiry = ""
	return s.savePeer(pubkey, peer)
}

func (s *LevelDBPeerStore) GetAllPubkeys() ([]PublicKey, error) {
	iter := s.DB.NewIterator(util.BytesPrefix([]byte("pubkey:")), nil)
	var pubkeys []PublicKey = nil
	for iter.Next() {
		pubkeyStr := string(iter.Key())[7:]
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
	if peer.IP == "" {
		highestIP, err := s.DB.Get([]byte(HighestIPKey), nil)
		if err == leveldb.ErrNotFound {
			highestIP = net.ParseIP("10.0.0.1")
		} else if err != nil {
			return nil, err
		}
		// TODO: cleverer way to allocate IP addresses
		ipInt := binary.BigEndian.Uint32(highestIP[12:])
		newPInt := ipInt + 1
		newIPBytes := [4]byte{}
		binary.BigEndian.PutUint32(newIPBytes[:], newPInt)
		newIP := net.IPv4(newIPBytes[0], newIPBytes[1], newIPBytes[2], newIPBytes[3])
		if err := s.DB.Put([]byte(HighestIPKey), newIP[:], nil); err != nil {
			return nil, err
		}
		peer.IP = newIP.String()
		if err := s.savePeer(pubkey, peer); err != nil {
			return nil, err
		}
		// TODO: check 10.0.0.0 namespace overflow
	}
	peerIP := net.ParseIP(peer.IP)
	return &peerIP, nil
}
