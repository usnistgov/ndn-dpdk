package ringbuffer_test

import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

func TestRing(t *testing.T) {
	assert, require := makeAR(t)

	r, e := ringbuffer.New("TestRing", 4, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	defer r.Close()

	assert.Equal(0, r.CountInUse())
	assert.Equal(3, r.CountAvailable())
	assert.Equal(r.CountAvailable(), r.GetCapacity())

	output := make([]unsafe.Pointer, 3)
	assert.Equal(0, r.Dequeue(output[:2]))

	input := []unsafe.Pointer{unsafe.Pointer(uintptr(9971)), unsafe.Pointer(uintptr(3087))}
	assert.Equal(2, r.Enqueue(input))
	assert.Equal(2, r.CountInUse())
	assert.Equal(1, r.CountAvailable())

	input = []unsafe.Pointer{unsafe.Pointer(uintptr(2776)), unsafe.Pointer(uintptr(1876))}
	assert.Equal(1, r.Enqueue(input))
	assert.Equal(3, r.CountInUse())
	assert.Equal(0, r.CountAvailable())

	assert.Equal(1, r.Dequeue(output[:1]))
	assert.Equal(unsafe.Pointer(uintptr(9971)), output[0])
	assert.Equal(2, r.CountInUse())
	assert.Equal(1, r.CountAvailable())

	assert.Equal(2, r.Dequeue(output[:3]))
	assert.Equal(unsafe.Pointer(uintptr(3087)), output[0])
	assert.Equal(unsafe.Pointer(uintptr(2776)), output[1])
	assert.Equal(0, r.CountInUse())
	assert.Equal(3, r.CountAvailable())
}

func TestCapacity(t *testing.T) {
	assert, require := makeAR(t)

	r, e := ringbuffer.New("TestRing-1", -1, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	assert.Equal(63, r.GetCapacity())
	defer r.Close()

	r, e = ringbuffer.New("TestRing129", 129, eal.NumaSocket{}, ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti)
	require.NoError(e)
	assert.Equal(255, r.GetCapacity())
	defer r.Close()
}
