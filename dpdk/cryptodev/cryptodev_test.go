package cryptodev_test

import (
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestCryptoDev(t *testing.T) {
	assert, require := makeAR(t)

	var cfg cryptodev.VDevConfig
	cfg.NQueuePairs = 2
	cd, e := cryptodev.CreateVDev(cfg)
	require.NoError(e)
	defer cd.Close()

	qps := cd.QueuePairs()
	require.Len(qps, 2)

	inputs := make([][]byte, 3)
	inputs[0] = make([]byte, 31)
	inputs[1] = make([]byte, 128)
	inputs[2] = make([]byte, 257)

	expects := make([][sha256.Size]byte, len(inputs))
	outputs := make([]*byte, len(inputs))
	ops := make([]*cryptodev.Op, len(inputs))

	for i := range inputs {
		rand.Read(inputs[i])
		expects[i] = sha256.Sum256(inputs[i])
		outputs[i] = eal.Zmalloc[byte]("", sha256.Size, eal.NumaSocket{})
		defer eal.Free(outputs[i])
		ops[i] = eal.Zmalloc[cryptodev.Op]("CryptoOp", unsafe.Sizeof(cryptodev.Op{}), eal.NumaSocket{})
		defer eal.Free(ops[i])
		qp := qps[0]
		if i == 2 {
			qp = qps[1]
		}
		qp.PrepareSha256(ops[i], makePacket(inputs[i]), 0, len(inputs[i]), unsafe.Pointer(outputs[i]))
		assert.EqualValues(cryptodev.OpStatusNew, ops[i].Status())
	}

	assert.Equal(2, qps[0].EnqueueBurst(ops[:2]))
	assert.Equal(1, qps[1].EnqueueBurst(ops[2:]))

	ops = make(cryptodev.OpVector, 2)
	require.Equal(1, qps[1].DequeueBurst(ops))
	assert.EqualValues(cryptodev.OpStatusSuccess, ops[0].Status())
	assert.Equal(expects[2][:], unsafe.Slice(outputs[2], sha256.Size))

	require.Equal(2, qps[0].DequeueBurst(ops))
	assert.EqualValues(cryptodev.OpStatusSuccess, ops[0].Status())
	assert.Equal(expects[0][:], unsafe.Slice(outputs[0], sha256.Size))
	assert.EqualValues(cryptodev.OpStatusSuccess, ops[1].Status())
	assert.Equal(expects[1][:], unsafe.Slice(outputs[1], sha256.Size))
	assert.Equal(0, qps[0].DequeueBurst(ops))
}
