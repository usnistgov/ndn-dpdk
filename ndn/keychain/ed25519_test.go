package keychain_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func TestEd25519Signing(t *testing.T) {
	assert, require := makeAR(t)
	publicA, privA, e := ed25519.GenerateKey(nil)
	require.NoError(e)

	subjectName := ndn.ParseName("/K")
	_, e = keychain.NewEd25519PrivateKey(subjectName, privA)
	assert.Error(e)
	_, e = keychain.NewEd25519PublicKey(subjectName, publicA)
	assert.Error(e)

	keyNameA := keychain.ToKeyName(subjectName)
	pvtA, e := keychain.NewEd25519PrivateKey(keyNameA, privA)
	require.NoError(e)
	pubA, e := keychain.NewEd25519PublicKey(keyNameA, publicA)
	require.NoError(e)
	nameEqual(assert, keyNameA, pvtA)
	nameEqual(assert, keyNameA, pubA)

	pvtB, pubB, e := keychain.NewEd25519KeyPair(subjectName)
	require.NoError(e)
	nameEqual(assert, pvtB, pubB)

	checkKeyCertPair(t, an.SigEd25519, pvtA, pvtB, pubA, pubB)
}

func TestEd25519Verify(t *testing.T) {
	assert, require := makeAR(t)

	data := ndntestvector.Ed25519Demo()
	cert, e := keychain.CertFromData(data)
	require.NoError(e)
	assert.True(cert.SelfSigned())

	key := cert.PublicKey()
	nameEqual(assert, "/Ed25519-demo/KEY/5a615db7cf0603b5", key)

	assert.NoError(key.Verify(data))
}
