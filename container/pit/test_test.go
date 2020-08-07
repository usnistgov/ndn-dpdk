package pit_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibreplica"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
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
	ealtestenv.Init()
	os.Exit(m.Run())
}

var (
	makeAR          = testenv.MakeAR
	nameEqual       = ndntestenv.NameEqual
	makeInterest    = ndnitestenv.MakeInterest
	makeData        = ndnitestenv.MakeData
	makeNack        = ndnitestenv.MakeNack
	setActiveFwHint = ndnitestenv.SetActiveFwHint
	setPitToken     = ndnitestenv.SetPitToken
	setFace         = ndnitestenv.SetFace
	makeFibEntry    = fibtestenv.MakeEntry
)

type Fixture struct {
	Pcct       *pcct.Pcct
	Pit        *pit.Pit
	Fib        *fib.Fib
	FibReplica *fibreplica.Table
	FibEntry   *fibreplica.Entry
}

func NewFixture(pcctMaxEntries int) *Fixture {
	fixture := new(Fixture)
	var e error
	fixture.Pcct, e = pcct.New(pcct.Config{MaxEntries: pcctMaxEntries}, eal.NumaSocket{})
	if e != nil {
		panic(e)
	}
	fixture.Pit = pit.FromPcct(fixture.Pcct)

	fixture.Fib, e = fib.New(fibdef.Config{Capacity: 1023}, []fib.LookupThread{&fibtestenv.LookupThread{}})
	if e != nil {
		panic(e)
	}
	placeholderName := ndn.ParseName("/75f3c2eb-6147-4030-afbc-585b3ce876a9")
	if e = fixture.Fib.Insert(makeFibEntry(placeholderName, nil, 9999)); e != nil {
		panic(e)
	}
	fixture.FibReplica = fixture.Fib.Replica(eal.NumaSocket{})
	fixture.FibEntry = fixture.FibReplica.Lpm(placeholderName)

	return fixture
}

func (fixture *Fixture) Close() error {
	fixture.Fib.Close()
	return fixture.Pcct.Close()
}

// Return number of in-use entries in PCCT's underlying mempool.
func (fixture *Fixture) CountMpInUse() int {
	return fixture.Pcct.AsMempool().CountInUse()
}

// Insert a PIT entry.
// Returns the PIT entry.
// If CS entry is found, returns nil and frees interest.
func (fixture *Fixture) Insert(interest *ndni.Packet) *pit.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.FibEntry)
	if csEntry != nil {
		interest.Close()
		return nil
	}
	if pitEntry == nil {
		panic("Pit.Insert failed")
	}
	return pitEntry
}

func (fixture *Fixture) InsertFibEntry(name string, nexthop iface.ID) *fibreplica.Entry {
	n := ndn.ParseName(name)
	if e := fixture.Fib.Insert(fibtestenv.MakeEntry(n, nil, nexthop)); e != nil {
		panic(e)
	}
	return fixture.FibReplica.Lpm(n)
}
