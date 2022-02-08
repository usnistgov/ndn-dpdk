package iface_test

import (
	"fmt"
	"math/rand"
	"testing"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"go4.org/must"
)

type InputDemuxFixture struct {
	D *iface.InputDemux
	Q []*PktQueueFixture
}

func (fixture *InputDemuxFixture) SetDests(n int) {
	fixture.Q = make([]*PktQueueFixture, n)
	for i := range fixture.Q {
		q := NewPktQueueFixture()
		q.Q.Init(iface.PktQueueConfig{
			Capacity:     65536,
			DisableCoDel: true,
		}, eal.NumaSocket{})
		fixture.D.SetDest(i, q.Q)
		fixture.Q[i] = q
	}
}

func (fixture *InputDemuxFixture) DispatchInterests(prefix string, n int, suffix string) (pkts []*ndni.Packet) {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s/%d%s", prefix, i, suffix)
		pkt := ndnitestenv.MakeInterest(name)
		pkts = append(pkts, pkt)
		fixture.D.Dispatch(pkt)
	}
	return pkts
}

func (fixture *InputDemuxFixture) Counts() (cnt []int) {
	cnt = make([]int, len(fixture.Q))
	for i, q := range fixture.Q {
		cnt[i] = q.Q.Ring().CountInUse()
	}
	return cnt
}

func (fixture *InputDemuxFixture) Close() error {
	for _, q := range fixture.Q {
		must.Close(q)
	}
	eal.Free(fixture.D)
	return nil
}

func NewInputDemuxFixture() (fixture *InputDemuxFixture) {
	return &InputDemuxFixture{
		D: (*iface.InputDemux)(eal.Zmalloc("InputDemux", unsafe.Sizeof(iface.InputDemux{}), eal.NumaSocket{})),
	}
}

func testInputDemuxRoundRobin(t testing.TB, n int) {
	assert, _ := makeAR(t)

	fixture := NewInputDemuxFixture()
	defer fixture.Close()
	fixture.SetDests(n)
	fixture.D.InitRoundrobin(n)

	fixture.DispatchInterests("/I", 500, "")
	cnt := fixture.Counts()
	sum := 0
	for _, c := range cnt {
		sum += c
	}
	assert.Equal(500, sum)
	assert.InDelta(math.MinIntN(cnt...), math.MaxIntN(cnt...), 1.0)
}

func TestInputDemuxRoundRobin5(t *testing.T) {
	testInputDemuxRoundRobin(t, 5)
}

func TestInputDemuxRoundRobin8(t *testing.T) {
	testInputDemuxRoundRobin(t, 8)
}

func testInputDemuxGenericHash(t testing.TB, n int) {
	assert, _ := makeAR(t)

	fixture := NewInputDemuxFixture()
	defer fixture.Close()
	fixture.SetDests(n)
	fixture.D.InitGenericHash(n)

	fixture.DispatchInterests("/I", 500, "/32=x")
	pkts := fixture.DispatchInterests("/I", 500, "")
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

	theNdt := ndt.New(ndt.Config{PrefixLen: 2}, []eal.NumaSocket{eal.Sockets[0]})
	defer theNdt.Close()
	prefix := "/..."
	for {
		indexSet := make(map[uint64]bool)
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

	fixture := NewInputDemuxFixture()
	defer fixture.Close()
	fixture.SetDests(10)
	fixture.D.InitNdt(theNdt.Queriers()[0])

	fixture.D.Dispatch(ndnitestenv.MakeInterest(prefix))
	fixture.DispatchInterests(prefix, 8, "")
	fixture.DispatchInterests(prefix, 8, "/A")
	fixture.DispatchInterests(prefix, 7, "/B")

	assert.Equal([]int{3, 3, 3, 3, 3, 3, 3, 2, 0, 1}, fixture.Counts())
}

func TestInputDemuxToken(t *testing.T) {
	assert, _ := makeAR(t)

	fixture := NewInputDemuxFixture()
	defer fixture.Close()
	fixture.SetDests(11)
	fixture.D.InitToken(1)

	for i := byte(0); i < 200; i++ {
		fixture.D.Dispatch(ndnitestenv.MakeInterest(fmt.Sprintf("/I/%d", i),
			ndnitestenv.SetPitToken([]byte{i / 100, i % 100, 0xA1})))
	}
	fixture.D.Dispatch(ndnitestenv.MakeInterest("/I/X", ndnitestenv.SetPitToken([]byte{0xA2})))

	assert.Equal([]int{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}, fixture.Counts())
	assert.Equal(179, int(fixture.D.Counters().NDrops))
	for i := 0; i < 11; i++ {
		cnt := fixture.D.DestCounters(i)
		assert.Equal(2, int(cnt.NQueued))
		assert.Equal(0, int(cnt.NDropped))
	}
}
