package keychain

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/youmark/pkcs8"
)

// ImportSafeBag imports a private key from ndn-cxx exported credentials.
// https://named-data.net/doc/ndn-cxx/0.8.0/specs/safe-bag.html
func ImportSafeBag(wire, passphrase []byte) (pvt PrivateKey, cert *Certificate, e error) {
	var safeBagTLV tlv.Element
	if e = tlv.Decode(wire, &safeBagTLV); e != nil {
		return nil, nil, e
	} else if safeBagTLV.Type != an.TtSafeBag {
		return nil, nil, tlv.ErrType
	}

	var pvtKey any
	d := tlv.DecodingBuffer(safeBagTLV.Value)
	for _, de := range d.Elements() {
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
	default:
		e = errors.New("unsupported private key type")
	}
	if e != nil {
		return nil, nil, e
	}

	return pvt, cert, nil
}
