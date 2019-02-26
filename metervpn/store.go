package metervpn

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"time"
)

type AllowanceStore interface {
	AddDuration(pubkey PublicKey, duration time.Duration) (*time.Time, error)
	GetExpiry(pubkey PublicKey) (*time.Time, error)
	GetAllPubkeys() ([]PublicKey, error)
	DeletePubkey(pubkey PublicKey) error
}

type LevelDBAllowanceStore struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC3339

func keyForPubkey(pubkey PublicKey) []byte {
	return []byte(
		fmt.Sprintf("pubkey:%v", MarshalPublicKey(pubkey)),
	)
}

func (s *LevelDBAllowanceStore) AddDuration(pubkey PublicKey, duration time.Duration) (*time.Time, error) {
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
	expiryBytes, err := s.DB.Get(keyForPubkey(pubkey), nil)
	if err == leveldb.ErrNotFound {
		now := time.Now()
		return &now, nil
	} else if err != nil {
		return nil, err
	}
	expiry, err := time.Parse(TimeLayout, string(expiryBytes))
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
