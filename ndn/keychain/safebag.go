package keychain

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/youmark/pkcs8"
)

// ExportSafeBag exports a private key to ndn-cxx exported credentials.
// https://docs.named-data.net/ndn-cxx/0.8.1/specs/safe-bag.html
func ExportSafeBag(pvt PrivateKey, cert *Certificate, passphrase []byte) (wire []byte, e error) {
	p, ok := pvt.(*privateKey)
	if !ok {
		return nil, errors.New("unsupported private key type")
	}
	encryptedKey, e := pkcs8.MarshalPrivateKey(p.key, passphrase, nil)
	if e != nil {
		return nil, e
	}
	return tlv.Encode(tlv.TLVFrom(an.TtSafeBag,
		cert.Data(),
		tlv.TLVBytes(an.TtSafeBagEncryptedKey, encryptedKey),
	))
}

// ImportSafeBag imports a private key from ndn-cxx exported credentials.
// https://docs.named-data.net/ndn-cxx/0.8.1/specs/safe-bag.html
func ImportSafeBag(wire, passphrase []byte) (pvt PrivateKey, cert *Certificate, e error) {
	var safeBagTLV tlv.Element
	if e = tlv.Decode(wire, &safeBagTLV); e != nil {
		return nil, nil, e
	} else if safeBagTLV.Type != an.TtSafeBag {
		return nil, nil, tlv.ErrType
	}

	var pvtKey any
	d := tlv.DecodingBuffer(safeBagTLV.Value)
	for de := range d.IterElements() {
		switch de.Type {
		case an.TtData:
			var data ndn.Data
			if e := de.UnmarshalValue(&data); e != nil {
				return nil, nil, e
			}
			if cert, e = CertFromData(data); e != nil {
				return nil, nil, e
			}
		case an.TtSafeBagEncryptedKey:
			if pvtKey, e = pkcs8.ParsePKCS8PrivateKey(de.Value, passphrase); e != nil {
				return nil, nil, e
			}
		default:
			if de.IsCriticalType() {
				return nil, nil, tlv.ErrCritical
			}
		}
	}
	if e := d.ErrUnlessEOF(); e != nil {
		return nil, nil, e
	}

	keyName := ToKeyName(cert.Name())
	switch pvtKey := pvtKey.(type) {
	case *ecdsa.PrivateKey:
		pvt, e = NewECDSAPrivateKey(keyName, pvtKey)
	case *rsa.PrivateKey:
		pvt, e = NewRSAPrivateKey(keyName, pvtKey)
	case ed25519.PrivateKey:
		pvt, e = NewEd25519PrivateKey(keyName, pvtKey)
	default:
		e = errors.New("unsupported private key type")
	}
	if e != nil {
		return nil, nil, e
	}

	return pvt, cert, nil
}
