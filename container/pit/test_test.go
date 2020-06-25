package pit_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtest"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	mbuftestenv.Direct.Template.Update(pktmbuf.PoolConfig{Dataroom: 8000}) // needed for TestEntryLongName
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	nameEqual    = ndntestenv.NameEqual
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
	setActiveFH  = ndnitestenv.SetActiveFH
)

type Fixture struct {
	Pit *pit.Pit

	fibFixture    *fibtest.Fixture
	emptyStrategy strategycode.StrategyCode
	EmptyFibEntry *fib.Entry
}

func NewFixture(pcctMaxEntries int) (fixture *Fixture) {
	fixture = new(Fixture)

	pcctCfg := pcct.Config{MaxEntries: pcctMaxEntries}
	pcct, e := pcct.New("TestPcct", pcctCfg)
	if e != nil {
		panic(e)
	}

	fixture.Pit = pit.FromPcct(pcct)

	fixture.fibFixture = fibtest.NewFixture(2, 4, 1)
	fixture.emptyStrategy = strategycode.MakeEmpty("empty")
	fixture.EmptyFibEntry = new(fib.Entry)
	return fixture
}

func (fixture *Fixture) Close() error {
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
func (fixture *Fixture) Insert(interest *ndni.Interest) *pit.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.EmptyFibEntry)
	if csEntry != nil {
		ndnitestenv.ClosePacket(interest)
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
	return fixture.fibFixture.Fib.Find(ndn.ParseName(name))
}
