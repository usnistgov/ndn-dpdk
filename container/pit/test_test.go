package pit_test

import (
	"os"
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/fib/fibtest"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 8000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

type Fixture struct {
	Pit pit.Pit

	fibFixture    *fibtest.Fixture
	emptyStrategy strategycode.StrategyCode
	EmptyFibEntry *fib.Entry
}

func NewFixture(pcctMaxEntries int) (fixture *Fixture) {
	fixture = new(Fixture)

	pcctCfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: pcctMaxEntries,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}
	pcct, e := pcct.New(pcctCfg)
	if e != nil {
		panic(e)
	}

	fixture.Pit = pit.Pit{pcct}

	fixture.fibFixture = fibtest.NewFixture(2, 4, 1)
	fixture.emptyStrategy = strategycode.MakeEmpty("empty")
	fixture.EmptyFibEntry = new(fib.Entry)
	return fixture
}

func (fixture *Fixture) Close() error {
	strategycode.CloseAll()
	fixture.fibFixture.Close()
	return fixture.Pit.Pcct.Close()
}

// Return number of in-use entries in PCCT's underlying mempool.
func (fixture *Fixture) CountMpInUse() int {
	return fixture.Pit.GetMempool().CountInUse()
}

// Insert a PIT entry.
// Returns the PIT entry.
// If CS entry is found, returns nil and frees interest.
func (fixture *Fixture) Insert(interest *ndn.Interest) *pit.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.EmptyFibEntry)
	if csEntry != nil {
		ndntestutil.ClosePacket(interest)
		return nil
	}
	if pitEntry == nil {
		panic("Pit.Insert failed")
	}
	return pitEntry
}

func (fixture *Fixture) InsertFibEntry(name string, nexthop iface.FaceId) *fib.Entry {
	if _, e := fixture.fibFixture.Fib.Insert(fixture.fibFixture.MakeEntry(name,
		fixture.emptyStrategy, nexthop)); e != nil {
		panic(e)
	}
	return fixture.fibFixture.Fib.Find(ndn.MustParseName(name))
}
