package keychain_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/keychain"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	nameEqual    = ndntestenv.NameEqual
	nameIsPrefix = ndntestenv.NameIsPrefix
)

func checkKeyCertPair(t testing.TB, sigType uint32, pvtA, pvtB keychain.PrivateKey, pubA, pubB keychain.PublicKey) {
	assert, require := makeAR(t)

	pvtWireA, e := keychain.MarshalKey(pvtA)
	require.NoError(e)
	pvtA, e = keychain.UnmarshalKey(pvtWireA)
	require.NoError(e)
	nameEqual(assert, pvtA, pubA)

	nameEqual(assert, pvtB, pubB)
	certB, e := keychain.MakeCert(pubB, pvtA, keychain.MakeCertOptions{})
	require.NoError(e)
	certWireB, e := keychain.MarshalCert(certB)
	require.NoError(e)
	certB, e = keychain.UnmarshalCert(certWireB)
	require.NoError(e)
	signerB := pvtB.WithKeyLocator(certB.Name())

	var c ndntestenv.SignVerifyTester
	c.PvtA, c.PvtB, c.PubA, c.PubB = pvtA, signerB, pubA, pubB
	c.CheckInterest(t)
	c.CheckInterestParameterized(t)
	rec := c.CheckData(t)

	dataA := rec.PktA.(*ndn.Data)
	assert.EqualValues(sigType, dataA.SigInfo.Type)
	nameEqual(assert, pubA, dataA.SigInfo.KeyLocator)
	dataB := rec.PktB.(*ndn.Data)
	assert.EqualValues(sigType, dataB.SigInfo.Type)
	nameEqual(assert, certB, dataB.SigInfo.KeyLocator)
}
