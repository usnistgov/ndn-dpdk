package keychain

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding"
	"errors"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

const validityPeriodTimeFormat = "20060102T150405"

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
	key      PublicKey
}

// Name returns the certificate name.
func (cert Certificate) Name() ndn.Name {
	return cert.data.Name
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
func (cert Certificate) PublicKey() PublicKey {
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
	vp := data.SigInfo.FindExtension(an.TtValidityPeriod)
	if vp == nil {
		return nil, ErrValidityPeriod
	}
	if e := cert.validity.UnmarshalBinary(vp.Value); e != nil {
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

// MakeCertOptions contains arguments to MakeCert function.
type MakeCertOptions struct {
	IssuerID  ndn.NameComponent
	Version   ndn.NameComponent
	Freshness time.Duration
	Validity  ValidityPeriod
}

func (opts *MakeCertOptions) applyDefaults() {
	if !opts.IssuerID.Valid() {
		opts.IssuerID = ComponentDefaultIssuer
	}
	if !opts.Version.Valid() {
		opts.Version = makeVersionFromCurrentTime()
	}
	opts.Freshness = opts.Freshness.Truncate(time.Second)
	if opts.Freshness <= 0 {
		opts.Freshness = time.Hour
	}
	if !opts.Validity.Valid() {
		opts.Validity = MaxValidityPeriod
	}
}

// MakeCert generates a certificate of the given public key, signed by the given signer.
func MakeCert(pub PublicKey, signer ndn.Signer, opts MakeCertOptions) (cert *Certificate, e error) {
	opts.applyDefaults()

	name := pub.Name().Append(opts.IssuerID, opts.Version)

	spki, e := pub.SPKI()
	if e != nil {
		return nil, e
	}

	vpWire, _ := tlv.EncodeFrom(opts.Validity)
	var vp tlv.Element
	if e = tlv.Decode(vpWire, &vp); e != nil {
		return nil, e
	}

	data := ndn.MakeData(name, ndn.ContentType(an.ContentKey), opts.Freshness, spki)
	data.SigInfo = &ndn.SigInfo{}
	data.SigInfo.Extensions = append(data.SigInfo.Extensions, vp)
	if e = signer.Sign(&data); e != nil {
		return nil, e
	}
	return CertFromData(data)
}

// ValidityPeriod represents ValidityPeriod element in an NDN certificate.
type ValidityPeriod struct {
	NotBefore time.Time
	NotAfter  time.Time
}

// MaxValidityPeriod is a very long ValidityPeriod.
var MaxValidityPeriod = ValidityPeriod{time.Unix(540109800, 0), time.Unix(253402300799, 0)}

var (
	_ tlv.Fielder                = ValidityPeriod{}
	_ encoding.BinaryUnmarshaler = (*ValidityPeriod)(nil)
)

// Valid checks whether fields are valid.
func (vp ValidityPeriod) Valid() bool {
	return !vp.NotBefore.IsZero() && !vp.NotAfter.IsZero() && !vp.NotBefore.After(vp.NotAfter)
}

// Includes determines whether the given timestamp is within validity period.
func (vp ValidityPeriod) Includes(t time.Time) bool {
	t = t.Truncate(time.Second)
	return !t.Before(vp.NotBefore) && !t.After(vp.NotAfter)
}

// Field implements tlv.Fielder interface.
func (vp ValidityPeriod) Field() tlv.Field {
	if !vp.Valid() {
		return tlv.FieldError(ErrValidityPeriod)
	}
	return tlv.TLV(an.TtValidityPeriod,
		tlv.TLVBytes(an.TtNotBefore, []byte(vp.NotBefore.Format(validityPeriodTimeFormat))),
		tlv.TLVBytes(an.TtNotAfter, []byte(vp.NotAfter.Format(validityPeriodTimeFormat))),
	)
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
	if !vp.Valid() {
		return ErrValidityPeriod
	}
	return d.ErrUnlessEOF()
}

func parseValidityPeriodTime(value []byte) time.Time {
	t, e := time.Parse(validityPeriodTimeFormat, string(value))
	if e != nil {
		return time.Time{}
	}
	return t
}

func init() {
	ndn.RegisterSigInfoExtension(an.TtValidityPeriod)
}
