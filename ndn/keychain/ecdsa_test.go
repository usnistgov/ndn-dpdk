package keychain_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
)

func TestECDSASigning(t *testing.T) {
	assert, require := makeAR(t)
	privA, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(e)

	subjectName := ndn.ParseName("/K")
	_, e = keychain.NewECDSAPrivateKey(subjectName, privA)
	assert.Error(e)
	_, e = keychain.NewECDSAPublicKey(subjectName, &privA.PublicKey)
	assert.Error(e)

	keyNameA := keychain.ToKeyName(subjectName)
	pvtA, e := keychain.NewECDSAPrivateKey(keyNameA, privA)
	require.NoError(e)
	pubA, e := keychain.NewECDSAPublicKey(keyNameA, &privA.PublicKey)
	require.NoError(e)
	nameEqual(assert, keyNameA, pvtA)
	nameEqual(assert, keyNameA, pubA)

	pvtB, pubB, e := keychain.NewECDSAKeyPair(subjectName)
	require.NoError(e)
	nameEqual(assert, pvtB, pubB)
	certNameB := keychain.ToCertName(pvtB.Name())
	signerB := pvtB.WithKeyLocator(certNameB)

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

func TestECDSAVerify(t *testing.T) {
	assert, require := makeAR(t)

	cert, e := keychain.CertFromData(ndntestvector.TestbedRootX3())
	require.NoError(e)
	assert.True(cert.SelfSigned())

	key := cert.PublicKey()
	nameEqual(assert, "/ndn", cert.SubjectName())
	nameEqual(assert, "/ndn/KEY/%EC%F1L%8EQ%23%15%E0", key)
	nameEqual(assert, "/ndn/KEY/%EC%F1L%8EQ%23%15%E0/ndn/%FD%00%00%01u%E6%7F2%10", cert.Data())

	data := ndntestvector.TestbedNeu20201217()
	assert.NoError(key.Verify(data))

	validity := cert.Validity()
	assert.Equal(time.Date(2020, 11, 20, 16, 31, 37, 0, time.UTC), validity.NotBefore)
	assert.Equal(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), validity.NotAfter)
	assert.False(validity.Includes(time.Date(2020, 11, 20, 16, 31, 36, 999999999, time.UTC)))
	assert.True(validity.Includes(time.Date(2020, 11, 20, 16, 31, 37, 0, time.UTC)))
	assert.True(validity.Includes(time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC)))
	assert.False(validity.Includes(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))
}
