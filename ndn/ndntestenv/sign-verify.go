package ndntestenv

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// SignVerifyTester tests Signer and Verifier.
type SignVerifyTester struct {
	PvtA, PvtB ndn.Signer
	PubA, PubB ndn.Verifier
	SameAB     bool
}

// SignVerifyRecord contains signed packets.
type SignVerifyRecord struct {
	PktA ndn.SignableVerifiable
	PktB ndn.SignableVerifiable
}

// Check runs the test with given makePacket function.
func (c SignVerifyTester) Check(t testing.TB, makePacket func(name ndn.Name) ndn.SignableVerifiable) (record SignVerifyRecord) {
	assert, require := testenv.MakeAR(t)

	name := ndn.ParseName("/NAME")
	record.PktA, record.PktB = makePacket(name), makePacket(name)
	require.NoError(c.PvtA.Sign(record.PktA))
	require.NoError(c.PvtB.Sign(record.PktB))

	vAA := c.PubA.Verify(record.PktA)
	vAB := c.PubB.Verify(record.PktA)
	vBA := c.PubA.Verify(record.PktB)
	vBB := c.PubB.Verify(record.PktB)

	assert.NoError(vAA, "verify pktA with pubA")
	assert.NoError(vBB, "verify pktB with pubB")

	if !c.SameAB {
		assert.Error(vAB, "verify pktA with pubB")
		assert.Error(vBA, "verify pktB with pubA")
	}
	return
}

// CheckInterest runs the test with non-parameterized Interest packets.
func (c SignVerifyTester) CheckInterest(t testing.TB) (record SignVerifyRecord) {
	return c.Check(t, func(name ndn.Name) ndn.SignableVerifiable {
		interest := ndn.MakeInterest(name)
		return &interest
	})
}

// CheckInterestParameterized runs the test with parameterized Interest packets.
func (c SignVerifyTester) CheckInterestParameterized(t testing.TB) (record SignVerifyRecord) {
	return c.Check(t, func(name ndn.Name) ndn.SignableVerifiable {
		interest := ndn.MakeInterest(name, []byte{0xC0, 0xC1})
		return &interest
	})
}

// CheckData runs the test with Data packets.
func (c SignVerifyTester) CheckData(t testing.TB) (record SignVerifyRecord) {
	return c.Check(t, func(name ndn.Name) ndn.SignableVerifiable {
		data := ndn.MakeData(name, []byte{0xC0, 0xC1})
		return &data
	})
}
