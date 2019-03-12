package daemon

import (
	"encoding/base64"
	"errors"
)

const PublicKeySize = 32

type PublicKey = [PublicKeySize]byte

func UnmarshalPublicKey(base64Pubkey string) (*PublicKey, error) {
	pubkeyBytes, err := base64.StdEncoding.DecodeString(base64Pubkey)
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
	return base64.StdEncoding.EncodeToString(pubkey[:])
}
