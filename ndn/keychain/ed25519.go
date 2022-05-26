package keychain

import (
	"crypto/ed25519"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// NewEd25519PrivateKey creates a private key for SigEd25519 signature type.
func NewEd25519PrivateKey(keyName ndn.Name, key ed25519.PrivateKey) (PrivateKey, error) {
	return newPrivateKey(an.SigEd25519, keyName, key, func(input []byte) (sig []byte, e error) {
		return ed25519.Sign(key, input), nil
	})
}

// NewEd25519PublicKey creates a public key for SigEd25519 signature type.
func NewEd25519PublicKey(keyName ndn.Name, key ed25519.PublicKey) (PublicKey, error) {
	return newPublicKey(an.SigEd25519, keyName, key, func(input, sig []byte) error {
		if ok := ed25519.Verify(key, input, sig); !ok {
			return ndn.ErrSigValue
		}
		return nil
	})
}

// NewEd25519KeyPair creates a key pair for SigEd25519 signature type.
func NewEd25519KeyPair(name ndn.Name) (PrivateKey, PublicKey, error) {
	keyName := ToKeyName(name)
	public, priv, e := ed25519.GenerateKey(nil)
	if e != nil {
		return nil, nil, e
	}
	pvt, e := NewEd25519PrivateKey(keyName, priv)
	if e != nil {
		return nil, nil, e
	}
	pub, e := NewEd25519PublicKey(keyName, public)
	if e != nil {
		return nil, nil, e
	}
	return pvt, pub, e
}
