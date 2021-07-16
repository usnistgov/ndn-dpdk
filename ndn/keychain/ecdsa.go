package keychain

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// NewECDSAPrivateKey creates a private key for SigSha256WithEcdsa signature type.
func NewECDSAPrivateKey(name ndn.Name, key *ecdsa.PrivateKey) (PrivateKeyKeyLocatorChanger, error) {
	if !IsKeyName(name) {
		return nil, ErrKeyName
	}
	var pvt ecdsaPrivateKey
	pvt.name = name
	pvt.key = key
	return &pvt, nil
}

// NewECDSAPublicKey creates a public key for SigSha256WithEcdsa signature type.
func NewECDSAPublicKey(name ndn.Name, key *ecdsa.PublicKey) (PublicKey, error) {
	if !IsKeyName(name) {
		return nil, ErrKeyName
	}
	var pub ecdsaPublicKey
	pub.name = name
	pub.key = key
	return &pub, nil
}

type ecdsaPrivateKey struct {
	name ndn.Name
	key  *ecdsa.PrivateKey
}

func (pvt *ecdsaPrivateKey) Name() ndn.Name {
	return pvt.name
}

func (pvt *ecdsaPrivateKey) Sign(packet ndn.Signable) error {
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

func (pvt *ecdsaPrivateKey) WithKeyLocator(klName ndn.Name) ndn.Signer {
	signer := *pvt
	signer.name = klName
	return &signer
}

type ecdsaPublicKey struct {
	name ndn.Name
	key  *ecdsa.PublicKey
}

func (pub *ecdsaPublicKey) Name() ndn.Name {
	return pub.name
}

func (pub *ecdsaPublicKey) Verify(packet ndn.Verifiable) error {
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
