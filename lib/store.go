package lib

import (
	"github.com/syndtr/goleveldb/leveldb"
	"time"
)

type Store struct {
	DB *leveldb.DB
}

const TimeLayout = time.RFC1123

func (s *Store) AddDuration(pubkey PublicKey, duration time.Duration) (*time.Time, error) {
	expiry, err := s.GetExpiry(pubkey)
	if err != nil {
		return nil, err
	}
	newExpiry := expiry.Add(duration)
	if err := s.DB.Put(pubkey[:], []byte(newExpiry.Format(TimeLayout)), nil); err != nil {
		return nil, err
	}
	return &newExpiry, nil
}

func (s *Store) GetExpiry(pubkey PublicKey) (*time.Time, error) {
	expiryBytes, err := s.DB.Get(pubkey[:], nil)
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

func (s *Store) GetTimeLeft(pubkey PublicKey) (*time.Duration, error) {
	expiry, err := s.GetExpiry(pubkey)
	if err != nil {
		return nil, err
	}
	timeLeft := expiry.Sub(time.Now())
	return &timeLeft, nil
}
