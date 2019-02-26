package lib

import (
	"encoding/hex"
	"github.com/syndtr/goleveldb/leveldb"
	"time"
)

type ExpiryStore interface {
	AddDuration(pubkey PublicKey, duration time.Duration) (*time.Time, error)
	GetExpiry(pubkey PublicKey) (*time.Time, error)
}

type LevelDBExpiryStore struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC1123

func keyForPubkey(pubkey PublicKey) []byte {
	return []byte(hex.EncodeToString(pubkey[:]))
}

func (s *LevelDBExpiryStore) AddDuration(pubkey PublicKey, duration time.Duration) (*time.Time, error) {
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

func (s *LevelDBExpiryStore) GetExpiry(pubkey PublicKey) (*time.Time, error) {
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

func (s *LevelDBExpiryStore) GetTimeLeft(pubkey PublicKey) (*time.Duration, error) {
	expiry, err := s.GetExpiry(pubkey)
	if err != nil {
		return nil, err
	}
	timeLeft := expiry.Sub(time.Now())
	return &timeLeft, nil
}
