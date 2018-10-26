package ndn_test

import (
	"crypto/sha256"
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		name      string
		freshness int
	}{
		{input: "0600", bad: true},                        // missing Name
		{input: "0604 meta=1400 content=1500", bad: true}, // missing Name
		{input: "0602 name=0700", name: "/"},
		{input: "0605 name=0703080141", name: "/A"},
		{input: "0615 name=0703080142 meta=140C (180102 fp=190201FF 1A03080142) content=1500", name: "/B", freshness: 0x01FF},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		e := pkt.ParseL3(theMp)
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L3PktType_Data, pkt.GetL3Type(), tt.input) {
				continue
			}
			data := pkt.AsData()
			assert.Implements((*ndn.IL3Packet)(nil), data)
			assert.Equal(tt.name, data.GetName().String(), tt.input)
			assert.EqualValues(tt.freshness, data.GetFreshnessPeriod()/time.Millisecond, tt.input)
		}
	}
}

func TestDataDigest(t *testing.T) {
	assert, require := makeAR(t)

	cd, e := dpdk.NewOpensslCryptoDev("", 1, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer cd.Close()
	qp, ok := cd.GetQueuePair(0)
	require.True(ok)
	mp, e := dpdk.NewCryptoOpPool("MP-CryptoOp", 255, 5, 0, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer mp.Close()

	names := []string{
		"/",
		"/A",
		"/B",
		"/C",
	}
	inputs := make([]*ndn.Data, 4)
	for i, name := range names {
		inputs[i] = ndntestutil.MakeData(name)
	}

	ops := make([]dpdk.CryptoOp, 4)
	require.NoError(mp.AllocBulk(dpdk.CRYPTO_OP_SYM, ops))
	for i, data := range inputs {
		assert.Nil(data.GetDigest())
		data.DigestPrepare(ops[i])
	}
	assert.Equal(4, qp.EnqueueBurst(ops))

	assert.Equal(4, qp.DequeueBurst(ops))
	for i, op := range ops {
		data, e := ndn.DataDigest_Finish(op)
		assert.NoError(e)
		if assert.NotNil(data) {
			assert.Equal(names[i], data.GetName().String())

			expectedDigest := sha256.Sum256(data.GetPacket().AsDpdkPacket().ReadAll())
			assert.Equal(expectedDigest[:], data.GetDigest())
		}
	}
}
