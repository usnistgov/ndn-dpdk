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
func NewRSAPrivateKey(name ndn.Name, key *rsa.PrivateKey) (PrivateKeyKeyLocatorChanger, error) {
	if !IsKeyName(name) {
		return nil, ErrKeyName
	}
	var pvt rsaPrivateKey
	pvt.name = name
	pvt.key = key
	return &pvt, nil
}

// NewRSAPublicKey creates a public key for SigSha256WithRsa signature type.
func NewRSAPublicKey(name ndn.Name, key *rsa.PublicKey) (PublicKey, error) {
	if !IsKeyName(name) {
		return nil, ErrKeyName
	}
	var pub rsaPublicKey
	pub.name = name
	pub.key = key
	return &pub, nil
}

type rsaPrivateKey struct {
	name ndn.Name
	key  *rsa.PrivateKey
}

func (pvt *rsaPrivateKey) Name() ndn.Name {
	return pvt.name
}

func (pvt *rsaPrivateKey) Sign(packet ndn.Signable) error {
	return packet.SignWith(func(name ndn.Name, si *ndn.SigInfo) (ndn.LLSign, error) {
		si.Type = an.SigSha256WithRsa
		si.KeyLocator = ndn.KeyLocator{
			Name: pvt.name,
		}
		return func(input []byte) (sig []byte, e error) {
			h := sha256.Sum256(input)
			return rsa.SignPKCS1v15(rand.Reader, pvt.key, crypto.SHA256, h[:])
		}, nil
	})
}

func (pvt *rsaPrivateKey) WithKeyLocator(klName ndn.Name) ndn.Signer {
	signer := *pvt
	signer.name = klName
	return &signer
}

type rsaPublicKey struct {
	name ndn.Name
	key  *rsa.PublicKey
}

func (pub *rsaPublicKey) Name() ndn.Name {
	return pub.name
}

func (pub *rsaPublicKey) Verify(packet ndn.Verifiable) error {
	return packet.VerifyWith(func(name ndn.Name, si ndn.SigInfo) (ndn.LLVerify, error) {
		if si.Type != an.SigSha256WithRsa {
			return nil, ndn.ErrSigType
		}
		return func(input, sig []byte) error {
			h := sha256.Sum256(input)
			return rsa.VerifyPKCS1v15(pub.key, crypto.SHA256, h[:], sig)
		}, nil
	})
}
