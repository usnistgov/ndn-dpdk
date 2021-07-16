package keychain_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func TestRSASigning(t *testing.T) {
	assert, require := makeAR(t)
	privA, e := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(e)
	privB, e := rsa.GenerateKey(rand.Reader, 2048)
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

	keyNameB := keychain.ToKeyName(subjectName)
	pvtB, e := keychain.NewRSAPrivateKey(keyNameB, privB)
	require.NoError(e)
	certNameB := keychain.ToCertName(keyNameB)
	signerB := pvtB.WithKeyLocator(certNameB)
	pubB, e := keychain.NewRSAPublicKey(keyNameB, &privB.PublicKey)
	require.NoError(e)

	var c ndntestenv.SignVerifyTester
	c.PvtA, c.PvtB, c.PubA, c.PubB = pvtA, signerB, pubA, pubB
	c.CheckInterest(t)
	c.CheckInterestParameterized(t)
	rec := c.CheckData(t)

	dataA := rec.PktA.(*ndn.Data)
	assert.EqualValues(an.SigSha256WithRsa, dataA.SigInfo.Type)
	nameEqual(assert, keyNameA, dataA.SigInfo.KeyLocator)
	dataB := rec.PktB.(*ndn.Data)
	assert.EqualValues(an.SigSha256WithRsa, dataB.SigInfo.Type)
	nameEqual(assert, certNameB, dataB.SigInfo.KeyLocator)
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
