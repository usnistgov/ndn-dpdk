package ndntestenv

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// SignVerifyTester tests Signer and Verifier.
type SignVerifyTester struct {
	MakePacket func(name ndn.Name) ndn.SignableVerifiable
	PvtA, PvtB ndn.Signer
	PubA, PubB ndn.Verifier
	SameAB     bool
}

// SignVerifyRecord contains signed packets.
type SignVerifyRecord struct {
	PktA ndn.SignableVerifiable
	PktB ndn.SignableVerifiable
}

// Check runs the test with assigned MakePacket function.
func (c SignVerifyTester) Check(t testing.TB) (record SignVerifyRecord) {
	assert, require := testenv.MakeAR(t)

	name := ndn.ParseName("/NAME")
	record.PktA, record.PktB = c.MakePacket(name), c.MakePacket(name)
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
	c.MakePacket = func(name ndn.Name) ndn.SignableVerifiable {
		interest := ndn.MakeInterest(name)
		return &interest
	}
	return c.Check(t)
}

// CheckInterestParameterized runs the test with parameterized Interest packets.
func (c SignVerifyTester) CheckInterestParameterized(t testing.TB) (record SignVerifyRecord) {
	c.MakePacket = func(name ndn.Name) ndn.SignableVerifiable {
		interest := ndn.MakeInterest(name, []byte{0xC0, 0xC1})
		return &interest
	}
	return c.Check(t)
}

// CheckData runs the test with Data packets.
func (c SignVerifyTester) CheckData(t testing.TB) (record SignVerifyRecord) {
	c.MakePacket = func(name ndn.Name) ndn.SignableVerifiable {
		data := ndn.MakeData(name, []byte{0xC0, 0xC1})
		return &data
	}
	return c.Check(t)
}
