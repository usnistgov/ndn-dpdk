package ndn

import (
	"crypto/hmac"
	"crypto/sha256"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// LLSign is a low-level signing function.
type LLSign func(input []byte) (sig []byte, e error)

// LLVerify is a low-level verification function.
type LLVerify func(input, sig []byte) error

// Signable is a packet that can be signed.
type Signable interface {
	SignWith(signer func(name Name, si *SigInfo) (LLSign, error)) error
}

// Verifiable is a packet that can be verified.
type Verifiable interface {
	VerifyWith(verifier func(name Name, si SigInfo) (LLVerify, error)) error
}

// SignableVerifiable is both Signable and Verifiable.
type SignableVerifiable interface {
	Signable
	Verifiable
}

// Signer is high-level signer, such as a private key.
type Signer interface {
	Sign(packet Signable) error
}

// Verifier is high-level verifier, such as a public key.
type Verifier interface {
	Verify(packet Verifiable) error
}

// SignerVerifier is both Signer and Verifier.
type SignerVerifier interface {
	Signer
	Verifier
}

// DigestSigning implements Signer and Verifier for SigSha256 signature type.
var DigestSigning SignerVerifier = digestSigning{}

type digestSigning struct{}

func (digestSigning) Sign(packet Signable) error {
	return packet.SignWith(func(name Name, si *SigInfo) (LLSign, error) {
		si.Type = an.SigSha256
		si.KeyLocator = KeyLocator{}
		return func(input []byte) (sig []byte, e error) {
			h := sha256.Sum256(input)
			return h[:], nil
		}, nil
	})
}

func (digestSigning) Verify(packet Verifiable) error {
	return packet.VerifyWith(func(name Name, si SigInfo) (LLVerify, error) {
		if si.Type != an.SigSha256 {
			return nil, ErrSigType
		}
		return func(input, sig []byte) error {
			h := sha256.Sum256(input)
			if !hmac.Equal(sig, h[:]) {
				return ErrSigValue
			}
			return nil
		}, nil
	})
}

// NullSigner implements Signer for SigNull signature type.
var NullSigner Signer = nullSigner{}

type nullSigner struct{}

func (nullSigner) Sign(packet Signable) error {
	return packet.SignWith(func(name Name, si *SigInfo) (LLSign, error) {
		si.Type = an.SigNull
		si.KeyLocator = KeyLocator{}
		return func(input []byte) (sig []byte, e error) {
			return nil, nil
		}, nil
	})
}

func newNullSigInfo() *SigInfo {
	return &SigInfo{
		Type: an.SigNull,
	}
}

// NopVerifier is a Verifier that performs no verification.
var NopVerifier Verifier = nopVerifier{}

type nopVerifier struct{}

func (nopVerifier) Verify(packet Verifiable) error {
	return nil
}
