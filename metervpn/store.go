package metervpn

import (
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"time"
)

type AllowanceStore interface {
	AddAllowance(pubkey PublicKey, duration time.Duration) (*time.Time, error)
	GetExpiry(pubkey PublicKey) (*time.Time, error)
	GetIPAddress(pubkey PublicKey) (*net.IP, error)

	GetAllPubkeys() ([]PublicKey, error)
	DeletePubkey(pubkey PublicKey) error
}

type LevelDBPeerRecord struct {
	Expiry string `json:"expiry"`
	IP     string `json:"ip"`
}

type LevelDBAllowanceStore struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC3339

const HighestIPKey = "highestIP"

func keyForPubkey(pubkey PublicKey) []byte {
	return []byte(
		fmt.Sprintf("pubkey:%v", MarshalPublicKey(pubkey)),
	)
}

func (s *LevelDBAllowanceStore) saveRecord(pubkey PublicKey, record LevelDBPeerRecord) error {
	obj, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.DB.Put(keyForPubkey(pubkey), obj, nil)
}

func (s *LevelDBAllowanceStore) loadOrCreateRecord(pubkey PublicKey) (*LevelDBPeerRecord, error) {
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

func (s *LevelDBAllowanceStore) AddAllowance(pubkey PublicKey, duration time.Duration) (*time.Time, error) {
	expiry, err := s.GetExpiry(pubkey)
	if err != nil {
		return nil, err
	}
	newExpiry := expiry.Add(duration)
	if err := s.DB.Put(keyForPubkey(pubkey), []byte(newExpiry.Format(TimeLayout)), nil); err != nil {
		return nil, err
	}
	return &newExpiry, nil
}

func (s *LevelDBAllowanceStore) GetExpiry(pubkey PublicKey) (*time.Time, error) {
	record, err := s.loadOrCreateRecord(pubkey)
	if record.Expiry == "" {
		now := time.Now()
		return &now, nil
	} else if err != nil {
		return nil, err
	}
	expiry, err := time.Parse(TimeLayout, record.Expiry)
	if err != nil {
		return nil, err
	}
	return &expiry, nil
}

func (s *LevelDBAllowanceStore) DeletePubkey(pubkey PublicKey) error {
	return s.DB.Delete(keyForPubkey(pubkey), nil)
}

func (s *LevelDBAllowanceStore) GetAllPubkeys() ([]PublicKey, error) {
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

func (s *LevelDBAllowanceStore) GetIPAddress(pubkey PublicKey) (*net.IP, error) {
	highestIP, err := s.DB.Get([]byte(HighestIPKey), nil)
	if err == leveldb.ErrNotFound {
		highestIP = net.IPv4(10, 0, 0, 1)
	} else if err != nil {
		return nil, err
	}
	// TODO: cleverer way to allocate IP addresses

}
