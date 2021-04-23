package keychain_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
)

func TestIsKeyName(t *testing.T) {
	assert, _ := makeAR(t)
	assert.False(keychain.IsKeyName(ndn.ParseName("/owner")))
	assert.False(keychain.IsKeyName(ndn.ParseName("/owner/KEY")))
	assert.True(keychain.IsKeyName(ndn.ParseName("/owner/KEY/key-id")))
	assert.False(keychain.IsKeyName(ndn.ParseName("/owner/KEY/key-id/issuer-id")))
	assert.False(keychain.IsKeyName(ndn.ParseName("/owner/KEY/key-id/issuer-id/version")))
}

func TestIsCertName(t *testing.T) {
	assert, _ := makeAR(t)
	assert.False(keychain.IsCertName(ndn.ParseName("/owner")))
	assert.False(keychain.IsCertName(ndn.ParseName("/owner/KEY")))
	assert.False(keychain.IsCertName(ndn.ParseName("/owner/KEY/key-id")))
	assert.False(keychain.IsCertName(ndn.ParseName("/owner/KEY/key-id/issuer-id")))
	assert.True(keychain.IsCertName(ndn.ParseName("/owner/KEY/key-id/issuer-id/version")))
}

func TestToSubjectName(t *testing.T) {
	assert, _ := makeAR(t)
	nameEqual(assert, "/owner", keychain.ToSubjectName(ndn.ParseName("/owner")))
	nameEqual(assert, "/owner", keychain.ToSubjectName(ndn.ParseName("/owner/KEY/key-id")))
	nameEqual(assert, "/owner", keychain.ToSubjectName(ndn.ParseName("/owner/KEY/key-id/issuer-id/version")))
}

func TestToKeyName(t *testing.T) {
	assert, _ := makeAR(t)
	nameEqual(assert, "/owner/KEY/key-id", keychain.ToKeyName(ndn.ParseName("/owner/KEY/key-id")))
	nameEqual(assert, "/owner/KEY/key-id", keychain.ToKeyName(ndn.ParseName("/owner/KEY/key-id/issuer-id/version")))

	keyName := keychain.ToKeyName(ndn.ParseName("/owner"))
	assert.Len(keyName, 3)
	nameIsPrefix(assert, "/owner/KEY", keyName)
}

func TestToCertName(t *testing.T) {
	assert, _ := makeAR(t)
	nameEqual(assert, "/owner/KEY/key-id/issuer-id/version", keychain.ToCertName(ndn.ParseName("/owner/KEY/key-id/issuer-id/version")))

	certName := keychain.ToCertName(ndn.ParseName("/owner/KEY/key-id"))
	assert.Len(certName, 5)
	nameIsPrefix(assert, "/owner/KEY/key-id", certName)
	assert.True(certName[3].Equal(keychain.ComponentDefaultIssuer))
	assert.EqualValues(an.TtVersionNameComponent, certName[4].Type)

	certName = keychain.ToCertName(ndn.ParseName("/owner"))
	assert.Len(certName, 5)
	nameIsPrefix(assert, "/owner/KEY", certName)
	assert.True(certName[3].Equal(keychain.ComponentDefaultIssuer))
	assert.EqualValues(an.TtVersionNameComponent, certName[4].Type)
}
