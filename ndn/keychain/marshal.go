package keychain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// MarshalKey serializes a private key to an internal format.
func MarshalKey(key PrivateKey) ([]byte, error) {
	pkey, _ := key.(*privateKey)
	if pkey == nil {
		return nil, fmt.Errorf("unknown key type %T", key)
	}

	name, e := tlv.EncodeFrom(pkey.Name())
	if e != nil {
		return nil, e
	}

	pkcs8, e := x509.MarshalPKCS8PrivateKey(key.(*privateKey).key)
	if e != nil {
		return nil, e
	}

	return bytes.Join([][]byte{name, pkcs8}, nil), nil
}

// UnmarshalKey deserializes a private key from the result of MarshalKey.
func UnmarshalKey(wire []byte) (PrivateKey, error) {
	d := tlv.DecodingBuffer(wire)
	nameEle, e := d.Element()
	if e != nil {
		return nil, e
	}
	var name ndn.Name
	if e := nameEle.UnmarshalValue(&name); e != nil {
		return nil, e
	}

	pkcs8 := d.Rest()
	key, e := x509.ParsePKCS8PrivateKey(pkcs8)
	if e != nil {
		return nil, e
	}

	switch key := key.(type) {
	case *rsa.PrivateKey:
		return NewRSAPrivateKey(name, key)
	case *ecdsa.PrivateKey:
		return NewECDSAPrivateKey(name, key)
	case ed25519.PrivateKey:
		return NewEd25519PrivateKey(name, key)
	}
	return nil, fmt.Errorf("unknown private key type %T", key)
}

// MarshalCert serializes a certificate to an internal format.
func MarshalCert(cert *Certificate) ([]byte, error) {
	return tlv.EncodeFrom(cert.Data())
}

// UnmarshalCert deserializes a certificate from the result of MarshalCert.
func UnmarshalCert(wire []byte) (*Certificate, error) {
	var pkt ndn.Packet
	e := tlv.Decode(wire, &pkt)
	if e != nil {
		return nil, e
	}
	if pkt.Data == nil {
		return nil, errors.New("certificate is not Data")
	}
	return CertFromData(*pkt.Data)
}
