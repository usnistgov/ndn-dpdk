package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestCryptoDev(t *testing.T) {
	assert, require := makeAR(t)

	cd, e := dpdk.NewOpensslCryptoDev("", 2, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer cd.Close()

	qp0, ok := cd.GetQueuePair(0)
	require.True(ok)
	qp1, ok := cd.GetQueuePair(1)
	require.True(ok)
	_, ok = cd.GetQueuePair(2)
	require.False(ok)

	mp, e := dpdk.NewCryptoOpPool("MP-CryptoOp", 255, 5, 0, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer mp.Close()

	ops0 := make([]dpdk.CryptoOp, 2)
	require.NoError(mp.AllocBulk(dpdk.CRYPTO_OP_SYM, ops0))
	ops1 := make([]dpdk.CryptoOp, 1)
	require.NoError(mp.AllocBulk(dpdk.CRYPTO_OP_SYM, ops1))
	assert.Equal(dpdk.CRYPTO_OP_NEW, ops1[0].GetStatus())

	allocOutBuf := func() (out dpdk.Segment) {
		out = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket().GetFirstSegment()
		out.Append(make([]byte, 32))
		return out
	}

	out0 := allocOutBuf()
	ops0[0].PrepareSha256Digest(dpdktestenv.PacketFromHex("A0A1A2A3"), 0, 4, out0.GetData())
	out1 := allocOutBuf()
	ops0[1].PrepareSha256Digest(dpdktestenv.PacketFromHex("B0B1B2B3"), 0, 4, out1.GetData())
	out2 := allocOutBuf()
	ops1[0].PrepareSha256Digest(dpdktestenv.PacketFromHex("C0C1C2C3"), 0, 4, out2.GetData())

	assert.Equal(2, qp0.EnqueueBurst(ops0))
	assert.Equal(1, qp1.EnqueueBurst(ops1))

	ops := make([]dpdk.CryptoOp, 2)
	assert.Equal(1, qp1.DequeueBurst(ops))
	assert.Equal(dpdk.CRYPTO_OP_SUCCESS, ops[0].GetStatus())
	assert.Equal(dpdktestenv.BytesFromHex("72D2A70D03005439DE209BBE9FFC050FAFD891082E9F3150F05A61054D25990F"),
		out2.AsByteSlice())

	assert.Equal(2, qp0.DequeueBurst(ops))
	assert.Equal(dpdk.CRYPTO_OP_SUCCESS, ops[0].GetStatus())
	assert.Equal(dpdktestenv.BytesFromHex("73B92B68882B199971462A2614C6691CBA581DA958740466030A64CE7DE66ED3"),
		out0.AsByteSlice())
	assert.Equal(dpdk.CRYPTO_OP_SUCCESS, ops[1].GetStatus())
	assert.Equal(dpdktestenv.BytesFromHex("A6662B764A4468DF70CA2CAD1B17DA26C62E53439DA8E4E8A80D9B91E59D09BA"),
		out1.AsByteSlice())
	assert.Equal(0, qp0.DequeueBurst(ops))

	assert.Equal(3, mp.CountInUse())
	ops0[0].Close()
	assert.Equal(2, mp.CountInUse())
}
