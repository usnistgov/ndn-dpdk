package cryptodev_test

import (
	"testing"

	"ndn-dpdk/dpdk/cryptodev"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
	"ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestCryptoDev(t *testing.T) {
	assert, require := makeAR(t)

	cd, e := cryptodev.SingleSegDrv.Create("", cryptodev.Config{NQueuePairs: 2}, eal.NumaSocket{})
	require.NoError(e)
	defer cd.Close()

	qp0 := cd.GetQueuePair(0)
	require.NotNil(qp0)
	qp1 := cd.GetQueuePair(1)
	require.NotNil(qp1)
	assert.Nil(cd.GetQueuePair(2))

	mp, e := cryptodev.NewOpPool("MP-CryptoOp", cryptodev.OpPoolConfig{Capacity: 255}, eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	ops0, e := mp.Alloc(cryptodev.OpSymmetric, 2)
	require.NoError(e)
	assert.Len(ops0, 2)
	ops1, e := mp.Alloc(cryptodev.OpSymmetric, 1)
	require.NoError(e)
	assert.Len(ops1, 1)
	assert.True(ops1[0].IsNew())

	allocOutBuf := func() (out *pktmbuf.Packet) {
		out = mbuftestenv.Direct.Alloc()
		out.Append(make([]byte, 32))
		return out
	}

	out0 := allocOutBuf()
	ops0[0].PrepareSha256Digest(mbuftestenv.MakePacket("A0A1A2A3"), 0, 4, out0.GetDataPtr())
	out1 := allocOutBuf()
	ops0[1].PrepareSha256Digest(mbuftestenv.MakePacket("B0B1B2B3"), 0, 4, out1.GetDataPtr())
	out2 := allocOutBuf()
	ops1[0].PrepareSha256Digest(mbuftestenv.MakePacket("C0C1C2C3"), 0, 4, out2.GetDataPtr())

	assert.Equal(2, qp0.EnqueueBurst(ops0))
	assert.Equal(1, qp1.EnqueueBurst(ops1))

	ops := make(cryptodev.OpVector, 2)
	assert.Equal(1, qp1.DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(mbuftestenv.BytesFromHex("72D2A70D03005439DE209BBE9FFC050FAFD891082E9F3150F05A61054D25990F"),
		out2.ReadAll())

	assert.Equal(2, qp0.DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(mbuftestenv.BytesFromHex("73B92B68882B199971462A2614C6691CBA581DA958740466030A64CE7DE66ED3"),
		out0.ReadAll())
	assert.True(ops[1].IsSuccess())
	assert.Equal(mbuftestenv.BytesFromHex("A6662B764A4468DF70CA2CAD1B17DA26C62E53439DA8E4E8A80D9B91E59D09BA"),
		out1.ReadAll())
	assert.Equal(0, qp0.DequeueBurst(ops))

	assert.Equal(3, mp.CountInUse())
	ops0[0].Close()
	assert.Equal(2, mp.CountInUse())
}
