// Package keychain implements signing and verification on NDN packets.
package keychain

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type namedSigner struct {
	sigType uint32
	klName  ndn.Name
	llSign  ndn.LLSign
}

// Sign implements ndn.Signer interface.
func (signer namedSigner) Sign(packet ndn.Signable) error {
	return packet.SignWith(func(name ndn.Name, si *ndn.SigInfo) (ndn.LLSign, error) {
		si.Type = signer.sigType
		si.KeyLocator = ndn.KeyLocator{Name: signer.klName}
		return signer.llSign, nil
	})
}

// PrivateKey represents a named private key.
type PrivateKey struct {
	namedSigner
}

var _ ndn.Signer = (*PrivateKey)(nil)

// Name returns key name.
func (pvt PrivateKey) Name() ndn.Name {
	return pvt.klName
}

// WithKeyLocator creates a new Signer that uses a different KeyLocator.
// This may be used to put certificate name in KeyLocator.
func (pvt PrivateKey) WithKeyLocator(klName ndn.Name) ndn.Signer {
	signer := pvt.namedSigner
	signer.klName = klName
	return &signer
}

// NewPrivateKey constructs a PrivateKey.
func NewPrivateKey(sigType uint32, keyName ndn.Name, llSign ndn.LLSign) (*PrivateKey, error) {
	if !IsKeyName(keyName) {
		return nil, ErrKeyName
	}
	return &PrivateKey{namedSigner{
		sigType: sigType,
		klName:  keyName,
		llSign:  llSign,
	}}, nil
}

// PublicKey represents a named public key.
type PublicKey struct {
	sigType  uint32
	keyName  ndn.Name
	llVerify ndn.LLVerify
	spki     func() ([]byte, error)
}

var _ ndn.Verifier = (*PublicKey)(nil)

// Name returns key name.
func (pub PublicKey) Name() ndn.Name {
	return pub.keyName
}

// Verify implements ndn.Verifier interface.
func (pub PublicKey) Verify(packet ndn.Verifiable) error {
	return packet.VerifyWith(func(name ndn.Name, si ndn.SigInfo) (ndn.LLVerify, error) {
		if si.Type != pub.sigType {
			return nil, ndn.ErrSigType
		}
		if !ToKeyName(si.KeyLocator.Name).Equal(ToKeyName(pub.keyName)) {
			return nil, ndn.ErrKeyLocator
		}
		return pub.llVerify, nil
	})
}

// SPKI returns public key in SubjectPublicKeyInfo format as used in NDN certificate.
func (pub PublicKey) SPKI() (spki []byte, e error) {
	return pub.spki()
}

// NewPublicKey constructs a PublicKey.
func NewPublicKey(sigType uint32, keyName ndn.Name, llVerify ndn.LLVerify, spki func() ([]byte, error)) (*PublicKey, error) {
	if !IsKeyName(keyName) {
		return nil, ErrKeyName
	}
	return &PublicKey{
		sigType:  sigType,
		keyName:  keyName,
		llVerify: llVerify,
		spki:     spki,
	}, nil
}
