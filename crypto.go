package udphp

import (
	"crypto/rand"
	"golang.org/x/crypto/curve25519"
)

func GenKeyPair() ([32]byte, [32]byte, error) {
	var pri [32]byte
	var pub [32]byte

	_, err := rand.Read(pri[:])
	if err != nil {
		return pri, pub, err
	}
	pri[0] &= 248
	pri[31] &= 127
	pri[31] |= 64

	curve25519.ScalarBaseMult(&pub, &pri)
	if err != nil {
		return pri, pub, err
	}

	return pri, pub, nil
}

func GenSharedSecret(selfPri, otherPub []byte) ([]byte, error) {
	var secret []byte
	secret, err := curve25519.X25519(selfPri, otherPub)
	if err != nil {
		return nil, err
	}

	return secret, nil
}