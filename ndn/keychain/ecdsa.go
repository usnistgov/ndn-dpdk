package keychain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// NewECDSAPrivateKey creates a private key for SigSha256WithEcdsa signature type.
func NewECDSAPrivateKey(keyName ndn.Name, key *ecdsa.PrivateKey) (*PrivateKey, error) {
	return NewPrivateKey(an.SigSha256WithEcdsa, keyName, func(input []byte) (sig []byte, e error) {
		h := sha256.Sum256(input)
		return ecdsa.SignASN1(rand.Reader, key, h[:])
	})
}

// NewECDSAPublicKey creates a public key for SigSha256WithEcdsa signature type.
func NewECDSAPublicKey(keyName ndn.Name, key *ecdsa.PublicKey) (*PublicKey, error) {
	return NewPublicKey(an.SigSha256WithEcdsa, keyName, func(input, sig []byte) error {
		h := sha256.Sum256(input)
		if ok := ecdsa.VerifyASN1(key, h[:], sig); !ok {
			return ndn.ErrSigValue
		}
		return nil
	})
}

// NewECDSAKeyPair creates a key pair for SigSha256WithEcdsa signature type.
func NewECDSAKeyPair(name ndn.Name) (*PrivateKey, *PublicKey, error) {
	keyName := ToKeyName(name)
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		return nil, nil, e
	}
	pvt, e := NewECDSAPrivateKey(keyName, key)
	if e != nil {
		return nil, nil, e
	}
	pub, e := NewECDSAPublicKey(keyName, &key.PublicKey)
	if e != nil {
		return nil, nil, e
	}
	return pvt, pub, e
}
