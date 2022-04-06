package iface_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"github.com/zyedidia/generic"
)

type InputDemuxFixture struct {
	t       testing.TB
	D       *iface.InputDemux
	Q       []*iface.PktQueue
	Rejects pktmbuf.Vector
}

func (fixture *InputDemuxFixture) SetDests(n int) {
	fixture.Q = make([]*iface.PktQueue, n)
	for i := range fixture.Q {
		q := NewPktQueueFixture(fixture.t, iface.PktQueueConfig{
			Capacity:     65536,
			DisableCoDel: true,
		}).Q
		fixture.D.SetDest(i, q)
		fixture.Q[i] = q
	}
}

func (fixture *InputDemuxFixture) Dispatch(pkt *ndni.Packet) (accepted bool) {
	accepted = fixture.D.Dispatch(pkt)
	if !accepted {
		fixture.Rejects = append(fixture.Rejects, pkt.Mbuf())
	}
	return
}

func (fixture *InputDemuxFixture) DispatchInterests(prefix string, n int, suffix string) (pkts []*ndni.Packet, nRejected int) {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s/%d%s", prefix, i, suffix)
		pkt := ndnitestenv.MakeInterest(name)
		pkts = append(pkts, pkt)
		if accepted := fixture.Dispatch(pkt); !accepted {
			nRejected++
		}
	}
	return
}

func (fixture *InputDemuxFixture) Counts() (cnt []int) {
	cnt = make([]int, len(fixture.Q))
	for i, q := range fixture.Q {
		cnt[i] = q.Ring().CountInUse()
	}
	return cnt
}

func NewInputDemuxFixture(t testing.TB) (fixture *InputDemuxFixture) {
	t.Cleanup(func() {
		eal.Free(fixture.D)
		fixture.Rejects.Close()
	})

	return &InputDemuxFixture{
		t: t,
		D: (*iface.InputDemux)(eal.Zmalloc("InputDemux", unsafe.Sizeof(iface.InputDemux{}), eal.NumaSocket{})),
	}
}

func TestInputDemuxDrop(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(2)
	// zero value of InputDemux drops packets

	fixture.DispatchInterests("/I", 500, "")
	assert.Len(fixture.Rejects, 500)
	assert.Equal([]int{0, 0}, fixture.Counts())
}

func TestInputDemuxFirst(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(2)
	fixture.D.InitFirst()

	fixture.DispatchInterests("/I", 500, "")
	assert.Empty(fixture.Rejects)
	assert.Equal([]int{500, 0}, fixture.Counts())
}

func testInputDemuxRoundRobin(t testing.TB, n int) {
	assert, _ := makeAR(t)
	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(n)
	fixture.D.InitRoundrobin(n)

	fixture.DispatchInterests("/I", 500, "")
	assert.Empty(fixture.Rejects)

	cnt := fixture.Counts()
	sum, min, max := 0, math.MaxInt, math.MinInt
	for _, c := range cnt {
		sum += c
		min = generic.Min(min, c)
		max = generic.Max(max, c)
	}
	assert.Equal(500, sum)
	assert.InDelta(min, max, 1.0)
}

func TestInputDemuxRoundRobin5(t *testing.T) {
	testInputDemuxRoundRobin(t, 5)
}

func TestInputDemuxRoundRobin8(t *testing.T) {
	testInputDemuxRoundRobin(t, 8)
}

func testInputDemuxGenericHash(t testing.TB, n int) {
	assert, _ := makeAR(t)
	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(n)
	fixture.D.InitGenericHash(n)

	fixture.DispatchInterests("/I", 500, "/32=x")
	pkts, _ := fixture.DispatchInterests("/I", 500, "")
	assert.Empty(fixture.Rejects)

	cnt := make([]int, n)
	for _, pkt := range pkts {
		hash := pkt.PName().ComputeHash()
		cnt[hash%uint64(n)] += 2
	}

	assert.Equal(cnt, fixture.Counts())
}

func TestInputDemuxGenericHash5(t *testing.T) {
	testInputDemuxGenericHash(t, 5)
}

func TestInputDemuxGenericHash8(t *testing.T) {
	testInputDemuxGenericHash(t, 8)
}

func TestInputDemuxNdt(t *testing.T) {
	assert, _ := makeAR(t)

	theNdt := ndt.New(ndt.Config{PrefixLen: 2}, nil)
	defer theNdt.Close()

	prefix := "/..."
	for {
		indexSet := map[uint64]bool{}
		for i := uint8(0); i < 10; i++ {
			name := prefix
			if i < 9 {
				name = fmt.Sprintf("%s/%d", prefix, i)
			}
			index := theNdt.IndexOfName(ndn.ParseName(name))
			indexSet[index] = true
			theNdt.Update(index, i)
		}
		if len(indexSet) == 10 {
			break
		}
		prefix = fmt.Sprintf("/%d", rand.Int())
	}

	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(10)
	ndq := fixture.D.InitNdt()
	ndq.Init(theNdt, eal.NumaSocket{})
	defer ndq.Clear(theNdt)

	fixture.Dispatch(ndnitestenv.MakeInterest(prefix))
	fixture.DispatchInterests(prefix, 8, "")
	fixture.DispatchInterests(prefix, 8, "/A")
	fixture.DispatchInterests(prefix, 7, "/B")
	assert.Empty(fixture.Rejects)

	assert.Equal([]int{3, 3, 3, 3, 3, 3, 3, 2, 0, 1}, fixture.Counts())
}

func TestInputDemuxToken(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(11)
	fixture.D.InitToken(1)

	for i := byte(0); i < 200; i++ {
		accepted := fixture.Dispatch(ndnitestenv.MakeInterest(fmt.Sprintf("/I/%d", i),
			ndnitestenv.SetPitToken([]byte{i / 100, i % 100, 0xA1})))
		if i%100 < 11 {
			assert.True(accepted, i)
		} else {
			assert.False(accepted, i)
		}
	}
	accepted := fixture.Dispatch(ndnitestenv.MakeInterest("/I/X", ndnitestenv.SetPitToken([]byte{0xA2})))
	assert.False(accepted)
	assert.Len(fixture.Rejects, 179)

	assert.Equal([]int{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}, fixture.Counts())
	assert.Equal(179, int(fixture.D.Counters().NDrops))
	for i := 0; i < 11; i++ {
		cnt := fixture.D.DestCounters(i)
		assert.Equal(2, int(cnt.NQueued))
		assert.Equal(0, int(cnt.NDropped))
	}
}
