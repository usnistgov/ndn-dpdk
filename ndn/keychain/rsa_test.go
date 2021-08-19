package keychain_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func TestRSASigning(t *testing.T) {
	assert, require := makeAR(t)
	privA, e := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(e)

	subjectName := ndn.ParseName("/K")
	_, e = keychain.NewRSAPrivateKey(subjectName, privA)
	assert.Error(e)
	_, e = keychain.NewRSAPublicKey(subjectName, &privA.PublicKey)
	assert.Error(e)

	keyNameA := keychain.ToKeyName(subjectName)
	pvtA, e := keychain.NewRSAPrivateKey(keyNameA, privA)
	require.NoError(e)
	pubA, e := keychain.NewRSAPublicKey(keyNameA, &privA.PublicKey)
	require.NoError(e)
	nameEqual(assert, keyNameA, pvtA)
	nameEqual(assert, keyNameA, pubA)

	pvtB, pubB, e := keychain.NewRSAKeyPair(subjectName)
	require.NoError(e)
	nameEqual(assert, pvtB, pubB)

	checkKeyCertPair(t, an.SigSha256WithRsa, pvtA, pvtB, pubA, pubB)
}

func TestRSAVerify(t *testing.T) {
	assert, require := makeAR(t)

	cert, e := keychain.CertFromData(ndntestvector.TestbedArizona20200301())
	require.NoError(e)
	assert.False(cert.SelfSigned())
	nameEqual(assert, "/ndn/KEY/e%9D%7F%A5%C5%81%10%7D", cert.Issuer())

	data := ndntestvector.TestbedShijunxiao20200301()
	assert.NoError(cert.PublicKey().Verify(data))
}
