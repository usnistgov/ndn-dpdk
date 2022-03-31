package ringbuffer_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

func TestCapacity(t *testing.T) {
	assert, require := makeAR(t)

	r, e := ringbuffer.New(-1, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	assert.Equal(255, r.Capacity())
	defer r.Close()

	r, e = ringbuffer.New(513, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	assert.Equal(1023, r.Capacity())
	defer r.Close()
}

func TestRing(t *testing.T) {
	assert, require := makeAR(t)

	r, e := ringbuffer.New(4, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	defer r.Close()

	assert.NotEmpty(r.String())
	assert.Equal(0, r.CountInUse())
	assert.Equal(3, r.CountAvailable())
	assert.Equal(r.CountAvailable(), r.Capacity())

	output := make([]uintptr, 3)
	assert.Equal(0, ringbuffer.Dequeue(r, output[:2]))

	input := []uintptr{9971, 3087}
	assert.Equal(2, ringbuffer.Enqueue(r, input))
	assert.Equal(2, r.CountInUse())
	assert.Equal(1, r.CountAvailable())

	input = []uintptr{2776, 1876}
	assert.Equal(1, ringbuffer.Enqueue(r, input))
	assert.Equal(3, r.CountInUse())
	assert.Equal(0, r.CountAvailable())

	assert.Equal(1, ringbuffer.Dequeue(r, output[:1]))
	assert.Equal(uintptr(9971), output[0])
	assert.Equal(2, r.CountInUse())
	assert.Equal(1, r.CountAvailable())

	assert.Equal(2, ringbuffer.Dequeue(r, output[:3]))
	assert.Equal(uintptr(3087), output[0])
	assert.Equal(uintptr(2776), output[1])
	assert.Equal(0, r.CountInUse())
	assert.Equal(3, r.CountAvailable())
}
