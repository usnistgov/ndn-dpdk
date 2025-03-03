package iface_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"github.com/zyedidia/generic/mapset"
)

type InputDemuxFixture struct {
	t         testing.TB
	D         *iface.InputDemux
	Q         []*iface.PktQueue
	ChunkSize int
	Rejects   pktmbuf.Vector
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

func (fixture *InputDemuxFixture) Dispatch(vec pktmbuf.Vector) (rejects []bool, nRejected int) {
	rejects = fixture.D.Dispatch(fixture.ChunkSize, vec)
	for i, rej := range rejects {
		if rej {
			fixture.Rejects = append(fixture.Rejects, vec[i])
			nRejected++
		}
	}
	return
}

func (fixture *InputDemuxFixture) DispatchInterests(prefix string, n int, suffix string) (pkts []*ndni.Packet, nRejected int) {
	pkts = make([]*ndni.Packet, n)
	vec := make(pktmbuf.Vector, n)
	for i := range n {
		name := fmt.Sprintf("%s/%d%s", prefix, i, suffix)
		pkts[i] = ndnitestenv.MakeInterest(name)
		vec[i] = pkts[i].Mbuf()
	}

	_, nRejected = fixture.Dispatch(vec)
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
		D: eal.Zmalloc[iface.InputDemux]("InputDemux", unsafe.Sizeof(iface.InputDemux{}), eal.NumaSocket{}),
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

	// ChunkSize matters in round-robin: each burst is sent to the same queue.
	fixture.ChunkSize = 2

	fixture.DispatchInterests("/I", 500, "")
	assert.Empty(fixture.Rejects)

	cnt := fixture.Counts()
	cntSum, cntMin, cntMax := 0, math.MaxInt, math.MinInt
	for _, c := range cnt {
		cntSum += c
		cntMin = min(cntMin, c)
		cntMax = max(cntMax, c)
	}
	assert.Equal(500, cntSum)
	assert.InDelta(cntMin, cntMax, float64(fixture.ChunkSize))
}

func TestInputDemuxRoundRobin(t *testing.T) {
	t.Run("div", func(t *testing.T) {
		testInputDemuxRoundRobin(t, 5)
	})
	t.Run("mask", func(t *testing.T) {
		testInputDemuxRoundRobin(t, 8)
	})
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

func TestInputDemuxGenericHash(t *testing.T) {
	t.Run("div", func(t *testing.T) {
		testInputDemuxGenericHash(t, 5)
	})
	t.Run("mask", func(t *testing.T) {
		testInputDemuxGenericHash(t, 8)
	})
}

func TestInputDemuxNdt(t *testing.T) {
	assert, _ := makeAR(t)

	theNdt := ndt.New(ndt.Config{PrefixLen: 2}, nil)
	defer theNdt.Close()

	prefix := "/..."
	for {
		indexSet := mapset.New[uint64]()
		for i := range uint8(10) {
			name := prefix
			if i < 9 {
				name = fmt.Sprintf("%s/%d", prefix, i)
			}
			index := theNdt.IndexOfName(ndn.ParseName(name))
			indexSet.Put(index)
			theNdt.Update(index, i)
		}
		if indexSet.Size() == 10 {
			break
		}
		prefix = fmt.Sprintf("/%d", rand.Int())
	}

	fixture := NewInputDemuxFixture(t)
	fixture.SetDests(10)
	ndq := fixture.D.InitNdt()
	ndq.Init(theNdt, eal.NumaSocket{})
	defer ndq.Clear(theNdt)

	fixture.Dispatch(pktmbuf.Vector{ndnitestenv.MakeInterest(prefix).Mbuf()})
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

	vec := make(pktmbuf.Vector, 201)
	for i := range vec {
		var token []byte
		switch i {
		case 200:
			token = []byte{0xA2}
		default:
			token = []byte{byte(i / 100), byte(i % 100), 0xA1}
		}
		vec[i] = ndnitestenv.MakeInterest(fmt.Sprintf("/I/%d", i), ndnitestenv.SetPitToken(token)).Mbuf()
	}

	rejects, _ := fixture.Dispatch(vec)
	for i, rejected := range rejects {
		switch {
		case i == 200: // token too short
			assert.True(rejected, i)
		case i%100 < 11: // accepted
			assert.False(rejected, i)
		default: // token value exceeds nDest
			assert.True(rejected, i)
		}
	}
	assert.Len(fixture.Rejects, 179)

	assert.Equal([]int{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}, fixture.Counts())
	assert.Equal(179, int(fixture.D.Counters().NDrops))
	for i := range 11 {
		cnt := fixture.D.DestCounters(i)
		assert.Equal(2, int(cnt.NQueued))
		assert.Equal(0, int(cnt.NDropped))
	}
}
