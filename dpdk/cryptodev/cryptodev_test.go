package cryptodev_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"go4.org/must"
)

func TestCryptoDev(t *testing.T) {
	assert, require := makeAR(t)

	var cfg cryptodev.VDevConfig
	cfg.NQueuePairs = 2
	cd, e := cryptodev.CreateVDev(cfg)
	require.NoError(e)
	defer cd.Close()

	qp := cd.QueuePairs()
	require.Len(qp, 2)

	mp, e := cryptodev.NewOpPool(cryptodev.OpPoolConfig{Capacity: 255}, eal.NumaSocket{})
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
		out = directMp.MustAlloc(1)[0]
		out.Append(make([]byte, 32))
		return out
	}

	out0 := allocOutBuf()
	ops0[0].PrepareSha256Digest(makePacket("A0A1A2A3"), 0, 4, out0.DataPtr())
	out1 := allocOutBuf()
	ops0[1].PrepareSha256Digest(makePacket("B0B1B2B3"), 0, 4, out1.DataPtr())
	out2 := allocOutBuf()
	ops1[0].PrepareSha256Digest(makePacket("C0C1C2C3"), 0, 4, out2.DataPtr())

	assert.Equal(2, qp[0].EnqueueBurst(ops0))
	assert.Equal(1, qp[1].EnqueueBurst(ops1))

	ops := make(cryptodev.OpVector, 2)
	assert.Equal(1, qp[1].DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(bytesFromHex("72D2A70D03005439DE209BBE9FFC050FAFD891082E9F3150F05A61054D25990F"),
		out2.Bytes())

	assert.Equal(2, qp[0].DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(bytesFromHex("73B92B68882B199971462A2614C6691CBA581DA958740466030A64CE7DE66ED3"),
		out0.Bytes())
	assert.True(ops[1].IsSuccess())
	assert.Equal(bytesFromHex("A6662B764A4468DF70CA2CAD1B17DA26C62E53439DA8E4E8A80D9B91E59D09BA"),
		out1.Bytes())
	assert.Equal(0, qp[0].DequeueBurst(ops))

	assert.Equal(3, mp.CountInUse())
	must.Close(ops0[0])
	assert.Equal(2, mp.CountInUse())
}
