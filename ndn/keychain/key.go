package keychain

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// PrivateKey represents a named private key.
type PrivateKey interface {
	ndn.Signer
	Name() ndn.Name
}

// PrivateKeyKeyLocatorChanger is a PrivateKey that can change KeyLocator.
type PrivateKeyKeyLocatorChanger interface {
	PrivateKey

	// WithKeyLocator creates a new Signer that uses a different KeyLocator.
	// This may be used to put certificate name in KeyLocator.
	WithKeyLocator(klName ndn.Name) ndn.Signer
}

// PublicKey represents a named public key.
type PublicKey interface {
	ndn.Verifier
	Name() ndn.Name
}

func init() {
	ndn.RegisterSigInfoExtension(an.TtValidityPeriod)
}
