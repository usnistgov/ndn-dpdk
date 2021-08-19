// Package keychain implements signing and verification on NDN packets.
package keychain

import (
	"crypto/x509"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

// PrivateKey represents a named private key.
type PrivateKey interface {
	ndn.Signer

	// Name returns key name.
	Name() ndn.Name

	// WithKeyLocator creates a new Signer that uses a different KeyLocator.
	// This may be used to put certificate name in KeyLocator.
	WithKeyLocator(klName ndn.Name) ndn.Signer
}

type namedSigner struct {
	sigType uint32
	klName  ndn.Name
	llSign  ndn.LLSign
}

func (signer namedSigner) Sign(packet ndn.Signable) error {
	return packet.SignWith(func(name ndn.Name, si *ndn.SigInfo) (ndn.LLSign, error) {
		si.Type = signer.sigType
		si.KeyLocator = ndn.KeyLocator{Name: signer.klName}
		return signer.llSign, nil
	})
}

type privateKey struct {
	namedSigner
	key interface{} // *rsa.PrivateKey or *ecdsa.PrivateKey
}

func (pvt privateKey) Name() ndn.Name {
	return pvt.klName
}

func (pvt privateKey) WithKeyLocator(klName ndn.Name) ndn.Signer {
	signer := pvt.namedSigner
	signer.klName = klName
	return &signer
}

func newPrivateKey(sigType uint32, keyName ndn.Name, key interface{}, llSign ndn.LLSign) (PrivateKey, error) {
	if !IsKeyName(keyName) {
		return nil, ErrKeyName
	}
	return &privateKey{
		namedSigner: namedSigner{
			sigType: sigType,
			klName:  keyName,
			llSign:  llSign,
		},
		key: key,
	}, nil
}

// PublicKey represents a named public key.
type PublicKey interface {
	ndn.Verifier

	// Name returns key name.
	Name() ndn.Name

	// SPKI returns public key in SubjectPublicKeyInfo format as used in NDN certificate.
	SPKI() (spki []byte, e error)
}

type publicKey struct {
	sigType  uint32
	keyName  ndn.Name
	key      interface{} // *rsa.PublicKey or *ecdsa.PublicKey
	llVerify ndn.LLVerify
}

func (pub publicKey) Name() ndn.Name {
	return pub.keyName
}

func (pub publicKey) Verify(packet ndn.Verifiable) error {
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

func (pub publicKey) SPKI() (spki []byte, e error) {
	return x509.MarshalPKIXPublicKey(pub.key)
}

func newPublicKey(sigType uint32, keyName ndn.Name, key interface{}, llVerify ndn.LLVerify) (PublicKey, error) {
	if !IsKeyName(keyName) {
		return nil, ErrKeyName
	}
	return &publicKey{
		sigType:  sigType,
		keyName:  keyName,
		key:      key,
		llVerify: llVerify,
	}, nil
}
