package keychain

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// NewRSAPrivateKey creates a private key for SigSha256WithRsa signature type.
func NewRSAPrivateKey(keyName ndn.Name, key *rsa.PrivateKey) (*PrivateKey, error) {
	return NewPrivateKey(an.SigSha256WithRsa, keyName, func(input []byte) (sig []byte, e error) {
		h := sha256.Sum256(input)
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	})
}

// NewRSAPublicKey creates a public key for SigSha256WithRsa signature type.
func NewRSAPublicKey(keyName ndn.Name, key *rsa.PublicKey) (*PublicKey, error) {
	return NewPublicKey(an.SigSha256WithRsa, keyName, func(input, sig []byte) error {
		h := sha256.Sum256(input)
		return rsa.VerifyPKCS1v15(key, crypto.SHA256, h[:], sig)
	})
}

// NewRSAKeyPair creates a key pair for SigSha256WithRsa signature type.
func NewRSAKeyPair(name ndn.Name) (*PrivateKey, *PublicKey, error) {
	keyName := ToKeyName(name)
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		return nil, nil, e
	}
	pvt, e := NewRSAPrivateKey(keyName, key)
	if e != nil {
		return nil, nil, e
	}
	pub, e := NewRSAPublicKey(keyName, &key.PublicKey)
	if e != nil {
		return nil, nil, e
	}
	return pvt, pub, e
}
