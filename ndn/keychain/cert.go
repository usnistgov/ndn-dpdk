package keychain

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Error conditions for certificate.
var (
	ErrCertContentType = errors.New("bad certificate ContentType")
	ErrValidityPeriod  = errors.New("bad ValidityPeriod")
	ErrX509PublicKey   = errors.New("bad X509PublicKey")
)

// Certificate represents an NDN certificate packet.
type Certificate struct {
	data     ndn.Data
	validity ValidityPeriod
	key      *PublicKey
}

// Data returns the certificate Data packet.
func (cert Certificate) Data() ndn.Data {
	return cert.data
}

// SubjectName returns the subject name.
func (cert Certificate) SubjectName() ndn.Name {
	return ToSubjectName(cert.data.Name)
}

// Issuer returns certificate issuer name, as it appears in the KeyLocator Name field.
// Returns nil if KeyLocator Name is absent.
func (cert Certificate) Issuer() ndn.Name {
	if cert.data.SigInfo == nil {
		return nil
	}
	return cert.data.SigInfo.KeyLocator.Name
}

// SelfSigned determines whether the certificate is self-signed.
func (cert Certificate) SelfSigned() bool {
	issuer := cert.Issuer()
	return issuer != nil && issuer.IsPrefixOf(cert.data.Name)
}

// Validity returns certificate ValidityPeriod.
func (cert Certificate) Validity() ValidityPeriod {
	return cert.validity
}

// PublicKey returns the enclosed public key.
func (cert Certificate) PublicKey() *PublicKey {
	return cert.key
}

// CertFromData parses a Data packet as certificate.
func CertFromData(data ndn.Data) (cert *Certificate, e error) {
	if !IsCertName(data.Name) {
		return nil, ErrCertName
	}
	if data.ContentType != an.ContentKey {
		return nil, ErrCertContentType
	}

	keyName := ToKeyName(data.Name)
	cert = &Certificate{data: data}

	if data.SigInfo == nil {
		return nil, ErrValidityPeriod
	}
	validityTlv := data.SigInfo.FindExtension(an.TtValidityPeriod)
	if validityTlv == nil {
		return nil, ErrValidityPeriod
	}
	if e := cert.validity.UnmarshalBinary(validityTlv.Value); e != nil {
		return nil, e
	}

	key, e := x509.ParsePKIXPublicKey(data.Content)
	if e != nil {
		return nil, ErrX509PublicKey
	}
	switch key := key.(type) {
	case *rsa.PublicKey:
		cert.key, _ = NewRSAPublicKey(keyName, key)
	case *ecdsa.PublicKey:
		cert.key, _ = NewECDSAPublicKey(keyName, key)
	}
	if cert.key == nil {
		return nil, ErrX509PublicKey
	}

	return cert, nil
}

// ValidityPeriod represents ValidityPeriod element in an NDN certificate.
type ValidityPeriod struct {
	NotBefore time.Time
	NotAfter  time.Time
}

// Includes determines whether the given timestamp is within validity period.
func (vp ValidityPeriod) Includes(t time.Time) bool {
	t = t.Truncate(time.Second)
	return !t.Before(vp.NotBefore) && !t.After(vp.NotAfter)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (vp *ValidityPeriod) UnmarshalBinary(wire []byte) (e error) {
	*vp = ValidityPeriod{}
	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtNotBefore:
			vp.NotBefore = parseValidityPeriodTime(de.Value)
		case an.TtNotAfter:
			vp.NotAfter = parseValidityPeriodTime(de.Value)
		default:
			if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	if vp.NotBefore.IsZero() || vp.NotAfter.IsZero() || vp.NotBefore.After(vp.NotAfter) {
		return ErrValidityPeriod
	}
	return d.ErrUnlessEOF()
}

func parseValidityPeriodTime(value []byte) time.Time {
	t, e := time.Parse("20060102T150405", string(value))
	if e != nil {
		return time.Time{}
	}
	return t
}

func init() {
	ndn.RegisterSigInfoExtension(an.TtValidityPeriod)
}
