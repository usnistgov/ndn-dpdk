package cryptodev_test

import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
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

	outPtr := eal.Zmalloc[byte]("cryptodev", 3*32, eal.NumaSocket{})
	defer eal.Free(outPtr)
	outSlice := unsafe.Slice(outPtr, 3*32)
	out0, out1, out2 := outSlice[0*32:1*32], outSlice[1*32:2*32], outSlice[2*32:3*32]

	ops0[0].PrepareSha256Digest(makePacket("A0A1A2A3"), 0, 4, unsafe.Pointer(&out0[0]))
	ops0[1].PrepareSha256Digest(makePacket("B0B1B2B3"), 0, 4, unsafe.Pointer(&out1[0]))
	ops1[0].PrepareSha256Digest(makePacket("C0C1C2C3"), 0, 4, unsafe.Pointer(&out2[0]))

	assert.Equal(2, qp[0].EnqueueBurst(ops0))
	assert.Equal(1, qp[1].EnqueueBurst(ops1))

	ops := make(cryptodev.OpVector, 2)
	assert.Equal(1, qp[1].DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(bytesFromHex("72D2A70D03005439DE209BBE9FFC050FAFD891082E9F3150F05A61054D25990F"), out2)

	assert.Equal(2, qp[0].DequeueBurst(ops))
	assert.True(ops[0].IsSuccess())
	assert.Equal(bytesFromHex("73B92B68882B199971462A2614C6691CBA581DA958740466030A64CE7DE66ED3"), out0)
	assert.True(ops[1].IsSuccess())
	assert.Equal(bytesFromHex("A6662B764A4468DF70CA2CAD1B17DA26C62E53439DA8E4E8A80D9B91E59D09BA"), out1)
	assert.Equal(0, qp[0].DequeueBurst(ops))

	assert.Equal(3, mp.CountInUse())
	must.Close(ops0[0])
	assert.Equal(2, mp.CountInUse())
}
