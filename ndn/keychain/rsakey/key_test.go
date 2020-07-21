package rsakey_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain/rsakey"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func TestSigning(t *testing.T) {
	assert, require := makeAR(t)
	privA, e := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(e)
	privB, e := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(e)

	subjectName := ndn.ParseName("/K")
	_, e = rsakey.NewPrivateKey(subjectName, privA)
	assert.Error(e)
	_, e = rsakey.NewPublicKey(subjectName, &privA.PublicKey)
	assert.Error(e)

	keyNameA := keychain.ToKeyName(subjectName)
	pvtA, e := rsakey.NewPrivateKey(keyNameA, privA)
	require.NoError(e)
	pubA, e := rsakey.NewPublicKey(keyNameA, &privA.PublicKey)
	require.NoError(e)
	nameEqual(assert, keyNameA, pvtA)
	nameEqual(assert, keyNameA, pubA)

	keyNameB := keychain.ToKeyName(subjectName)
	pvtB, e := rsakey.NewPrivateKey(keyNameB, privB)
	require.NoError(e)
	certNameB := keychain.ToCertName(keyNameB)
	signerB := pvtB.WithKeyLocator(certNameB)
	pubB, e := rsakey.NewPublicKey(keyNameB, &privB.PublicKey)
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

func TestVerify(t *testing.T) {
	assert, require := makeAR(t)
	cert := ndntestvector.TestbedArizona20200301()
	data := ndntestvector.TestbedShijunxiao20200301()

	rsaPublicKey, e := x509.ParsePKIXPublicKey(cert.Content)
	require.NoError(e)
	pub, e := rsakey.NewPublicKey(keychain.ToKeyName(cert.Name), rsaPublicKey.(*rsa.PublicKey))
	require.NoError(e)

	assert.NoError(pub.Verify(data))
}
