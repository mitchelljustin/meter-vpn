package daemon

import (
	"encoding/hex"
	"errors"
)

const PublicKeySize = 32

type PublicKey = [PublicKeySize]byte

func UnmarshalPublicKey(hexPubkey string) (*PublicKey, error) {
	pubkeyBytes, err := hex.DecodeString(hexPubkey)
	if err != nil {
		return nil, err
	}
	if len(pubkeyBytes) != PublicKeySize {
		return nil, errors.New("bad public key size")
	}
	var pubkey PublicKey
	copy(pubkey[:PublicKeySize], pubkeyBytes)
	return &pubkey, nil
}

func MarshalPublicKey(pubkey PublicKey) string {
	return hex.EncodeToString(pubkey[:])
}
