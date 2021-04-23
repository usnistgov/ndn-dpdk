package keychain

import (
	"errors"
	"math/rand"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Name components for certificate naming.
var (
	ComponentKEY           = ndn.ParseNameComponent("KEY")
	ComponentSelfIssuer    = ndn.ParseNameComponent("self")
	ComponentDefaultIssuer = ndn.ParseNameComponent("NDNgo-keychain")
)

// Error conditions for certificate naming.
var (
	ErrKeyName  = errors.New("bad key name")
	ErrCertName = errors.New("bad certificate name")
)

// IsKeyName determines if the input is a key name.
func IsKeyName(name ndn.Name) bool {
	return name.Get(-2).Equal(ComponentKEY)
}

// IsCertName determines if the input is a certificate name.
func IsCertName(name ndn.Name) bool {
	return name.Get(-4).Equal(ComponentKEY)
}

// ToSubjectName extracts subject name from subject name, key name, or certificate name.
func ToSubjectName(input ndn.Name) ndn.Name {
	switch {
	case IsKeyName(input):
		return input.GetPrefix(-2)
	case IsCertName(input):
		return input.GetPrefix(-4)
	}
	return input
}

// ToKeyName extracts or builds key name from subject name, key name, or certificate name.
// If the input is a subject name, the keyID component is randomly generated.
func ToKeyName(input ndn.Name) ndn.Name {
	switch {
	case IsKeyName(input):
		return input
	case IsCertName(input):
		return input.GetPrefix(-2)
	}

	keyID := makeRandomKeyID()
	return append(append(ndn.Name{}, input...), ComponentKEY, keyID)
}

// ToCertName extracts or builds certificate name from subject name, key name, or certificate name.
// If the input is a subject name, the keyID component is randomly generated.
// If the input is a subject name or key name, the issuerID is set to 'ndn-dpdk.go', and the version component is randomly generated.
func ToCertName(input ndn.Name) ndn.Name {
	switch {
	case IsCertName(input):
		return input
	case IsKeyName(input):
		version := makeVersionFromCurrentTime()
		return append(append(ndn.Name{}, input...), ComponentDefaultIssuer, version)
	}

	keyID := makeRandomKeyID()
	version := makeVersionFromCurrentTime()
	return append(append(ndn.Name{}, input...), ComponentKEY, keyID, ComponentDefaultIssuer, version)
}

// IsCertificate determines if the Data packet is a certificate.
func IsCertificate(data ndn.Data) bool {
	return data.ContentType == an.ContentKey && IsCertName(data.Name)
}

func makeRandomKeyID() ndn.NameComponent {
	value := make([]byte, 8)
	rand.Read(value)
	return ndn.MakeNameComponent(an.TtGenericNameComponent, value)
}

func makeVersionFromCurrentTime() (comp ndn.NameComponent) {
	now := time.Now().UnixNano() / int64(time.Microsecond/time.Nanosecond)
	return ndn.NameComponentFrom(an.TtVersionNameComponent, tlv.NNI(now))
}
