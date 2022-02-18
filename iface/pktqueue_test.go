package iface_test

import (
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
)

type PktQueueFixture struct {
	Q *iface.PktQueue
}

func (fixture *PktQueueFixture) PopMax(vec pktmbuf.Vector, now eal.TscTime) (count int, drop bool) {
	for {
		n, d := fixture.Q.Pop(vec[count:], now)
		count += n
		drop = drop || d
		if n == 0 || count == len(vec) {
			return
		}
	}
}

func NewPktQueueFixture(t testing.TB, cfg iface.PktQueueConfig) (fixture *PktQueueFixture) {
	_, require := makeAR(t)
	fixture = &PktQueueFixture{
		Q: (*iface.PktQueue)(eal.ZmallocAligned("PktQueue", unsafe.Sizeof(iface.PktQueue{}), 1, eal.NumaSocket{})),
	}
	t.Cleanup(func() {
		fixture.Q.Close()
		eal.Free(fixture.Q)
	})
	require.NoError(fixture.Q.Init(cfg, eal.NumaSocket{}))
	return
}

func gatherMbufPtrs(vec pktmbuf.Vector) (ptrs []unsafe.Pointer) {
	for _, m := range vec {
		ptrs = append(ptrs, m.Ptr())
	}
	return ptrs
}

func TestPktQueuePlain(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewPktQueueFixture(t, iface.PktQueueConfig{
		Capacity:     256,
		DisableCoDel: true,
	})

	vec, e := mbuftestenv.DirectMempool().Alloc(400)
	require.NoError(e)
	ptrs := gatherMbufPtrs(vec)

	assert.Equal(0, fixture.Q.Push(vec[0:100], eal.TscNow()))
	assert.Equal(0, fixture.Q.Push(vec[100:200], eal.TscNow()))
	if assert.Equal(45, fixture.Q.Push(vec[200:300], eal.TscNow())) {
		vec[255:300].Close()
	}

	deq := make(pktmbuf.Vector, 200)
	count, drop := fixture.PopMax(deq, eal.TscNow())
	if assert.Equal(200, count) {
		assert.Equal(ptrs[0:200], gatherMbufPtrs(deq))
	}
	assert.False(drop)
	deq.Close()

	assert.Equal(0, fixture.Q.Push(vec[300:400], eal.TscNow()))
	count, drop = fixture.PopMax(deq, eal.TscNow())
	if assert.Equal(155, count) {
		assert.Equal(ptrs[200:255], gatherMbufPtrs(deq[:55]))
		assert.Equal(ptrs[300:400], gatherMbufPtrs(deq[55:155]))
	}
	assert.False(drop)
	deq.Close()
}

func TestPktQueueDelay(t *testing.T) {
	assert, require := makeAR(t)
	const delay = 20 * time.Millisecond
	fixture := NewPktQueueFixture(t, iface.PktQueueConfig{
		DequeueBurstSize: 30,
		Delay:            nnduration.Nanoseconds(delay),
	})

	vec, e := mbuftestenv.DirectMempool().Alloc(50)
	require.NoError(e)
	ptrs := gatherMbufPtrs(vec)

	t0 := eal.TscNow()
	assert.Equal(0, fixture.Q.Push(vec, t0))

	deq := make(pktmbuf.Vector, 60)
	t1 := eal.TscNow()
	count2, drop2 := fixture.Q.Pop(deq, t1)
	t2 := eal.TscNow()
	count3, drop3 := fixture.Q.Pop(deq[30:], t2)
	t3 := eal.TscNow()
	count4, _ := fixture.Q.Pop(deq[50:], t3)

	if assert.Equal(30, count2) && assert.Equal(20, count3) && assert.Equal(0, count4) {
		assert.Equal(ptrs, gatherMbufPtrs(deq[:50]))
	}
	assert.False(drop2 || drop3)
	deq.Close()

	assert.Less(t1.Sub(t0), delay)
	assert.GreaterOrEqual(t2.Sub(t0), delay)
	assert.Less(t3.Sub(t2), delay)
}

func TestPktQueueCoDel(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewPktQueueFixture(t, iface.PktQueueConfig{})

	nEnq, nDeq, nDrop := 0, 0, 0
	enq := func(n int) {
		vec, e := mbuftestenv.DirectMempool().Alloc(n)
		require.NoError(e)
		nRej := fixture.Q.Push(vec, eal.TscNow())
		nEnq += n - nRej
		vec[n-nRej:].Close()
	}
	deq := func(n int) {
		vec := make(pktmbuf.Vector, n)
		count, drop := fixture.Q.Pop(vec, eal.TscNow())
		vec[:count].Close()
		nDeq += count
		if drop {
			nDrop++
		}
	}

	for i := 0; i < 500; i++ {
		enq(18)
		deq(20)
		time.Sleep(time.Millisecond)
	}
	assert.Equal(nEnq, nDeq)
	assert.Equal(nDrop, 0)

	nEnq, nDeq, nDrop = 0, 0, 0
	for i := 0; i < 500; i++ {
		enq(20)
		deq(18)
		time.Sleep(time.Millisecond)
	}
	assert.Greater(nDrop, 0)

	for i := 0; i < 500; i++ {
		enq(15)
		deq(20)
		time.Sleep(time.Millisecond)
	}
	deq(20)

	nDrop = 0
	for i := 0; i < 500; i++ {
		enq(18)
		deq(20)
		time.Sleep(time.Millisecond)
	}
	assert.Equal(nEnq, nDeq)
	assert.Equal(nDrop, 0)
}
