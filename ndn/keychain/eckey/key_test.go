package eckey_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain/eckey"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestSigning(t *testing.T) {
	assert, require := makeAR(t)
	privA, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(e)
	privB, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(e)

	subjectName := ndn.ParseName("/K")
	_, e = eckey.NewPrivateKey(subjectName, privA)
	assert.Error(e)
	_, e = eckey.NewPublicKey(subjectName, &privA.PublicKey)
	assert.Error(e)

	keyNameA := keychain.ToKeyName(subjectName)
	pvtA, e := eckey.NewPrivateKey(keyNameA, privA)
	require.NoError(e)
	pubA, e := eckey.NewPublicKey(keyNameA, &privA.PublicKey)
	require.NoError(e)
	nameEqual(assert, keyNameA, pvtA)
	nameEqual(assert, keyNameA, pubA)

	keyNameB := keychain.ToKeyName(subjectName)
	pvtB, e := eckey.NewPrivateKey(keyNameB, privB)
	require.NoError(e)
	certNameB := keychain.ToCertName(keyNameB)
	signerB := pvtB.WithKeyLocator(certNameB)
	pubB, e := eckey.NewPublicKey(keyNameB, &privB.PublicKey)
	require.NoError(e)

	var c ndntestenv.SignVerifyTester
	c.PvtA, c.PvtB, c.PubA, c.PubB = pvtA, signerB, pubA, pubB
	c.CheckInterest(t)
	c.CheckInterestParameterized(t)
	rec := c.CheckData(t)

	dataA := rec.PktA.(*ndn.Data)
	assert.EqualValues(an.SigSha256WithEcdsa, dataA.SigInfo.Type)
	nameEqual(assert, keyNameA, dataA.SigInfo.KeyLocator)
	dataB := rec.PktB.(*ndn.Data)
	assert.EqualValues(an.SigSha256WithEcdsa, dataB.SigInfo.Type)
	nameEqual(assert, certNameB, dataB.SigInfo.KeyLocator)
}

// TestVerify test case is absent due to lack of test vector.
// ndntestvector.TestbedRootV2() uses "specific curve" format that is unsupported by Go crypto/x509 library.
// See https://redmine.named-data.net/issues/5037
