// Package eckey implements SigSha256WithEcdsa signature type.
package eckey

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
)

// NewPrivateKey creates a private key for SigSha256WithEcdsa signature type.
func NewPrivateKey(name ndn.Name, key *ecdsa.PrivateKey) (keychain.PrivateKeyKeyLocatorChanger, error) {
	if !keychain.IsKeyName(name) {
		return nil, keychain.ErrKeyName
	}
	var pvt privateKey
	pvt.name = name
	pvt.key = key
	return &pvt, nil
}

// NewPublicKey creates a public key for SigSha256WithEcdsa signature type.
func NewPublicKey(name ndn.Name, key *ecdsa.PublicKey) (keychain.PublicKey, error) {
	if !keychain.IsKeyName(name) {
		return nil, keychain.ErrKeyName
	}
	var pub publicKey
	pub.name = name
	pub.key = key
	return &pub, nil
}

type privateKey struct {
	name ndn.Name
	key  *ecdsa.PrivateKey
}

func (pvt *privateKey) Name() ndn.Name {
	return pvt.name
}

func (pvt *privateKey) Sign(packet ndn.Signable) error {
	return packet.SignWith(func(name ndn.Name, si *ndn.SigInfo) (ndn.LLSign, error) {
		si.Type = an.SigSha256WithEcdsa
		si.KeyLocator = ndn.KeyLocator{
			Name: pvt.name,
		}
		return func(input []byte) (sig []byte, e error) {
			h := sha256.Sum256(input)
			return ecdsa.SignASN1(rand.Reader, pvt.key, h[:])
		}, nil
	})
}

func (pvt *privateKey) WithKeyLocator(klName ndn.Name) ndn.Signer {
	signer := *pvt
	signer.name = klName
	return &signer
}

type publicKey struct {
	name ndn.Name
	key  *ecdsa.PublicKey
}

func (pub *publicKey) Name() ndn.Name {
	return pub.name
}

func (pub *publicKey) Verify(packet ndn.Verifiable) error {
	return packet.VerifyWith(func(name ndn.Name, si ndn.SigInfo) (ndn.LLVerify, error) {
		if si.Type != an.SigSha256WithEcdsa {
			return nil, ndn.ErrSigType
		}
		return func(input, sig []byte) error {
			h := sha256.Sum256(input)
			if ok := ecdsa.VerifyASN1(pub.key, h[:], sig); !ok {
				return ErrVerification
			}
			return nil
		}, nil
	})
}

// ErrVerification represents a failure to verify a signature.
var ErrVerification = errors.New("ECDSA verification error")
