package cs_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/cs"
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
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()

	// fixture.Close() cannot release packet buffers, need a large mempool
	mbuftestenv.Direct.Template.Update(pktmbuf.PoolConfig{Capacity: 65535})

	os.Exit(m.Run())
}

var (
	makeAR          = testenv.MakeAR
	nameEqual       = ndntestenv.NameEqual
	makeInterest    = ndnitestenv.MakeInterest
	makeData        = ndnitestenv.MakeData
	setActiveFwHint = ndnitestenv.SetActiveFwHint
	setPitToken     = ndnitestenv.SetPitToken
	setFace         = ndnitestenv.SetFace
	makeFibEntry    = fibtestenv.MakeEntry
)

type Fixture struct {
	Pcct       *pcct.Pcct
	Cs         *cs.Cs
	Pit        *pit.Pit
	Fib        *fib.Fib
	FibReplica *fibreplica.Table
	FibEntry   *fibreplica.Entry
}

func NewFixture(cfg pcct.Config) *Fixture {
	cfg.MaxEntries = 4095
	if cfg.CsCapMd == 0 {
		cfg.CsCapMd = 200
	}
	if cfg.CsCapMi == 0 {
		cfg.CsCapMd = 200
	}

	fixture := new(Fixture)
	var e error
	fixture.Pcct, e = pcct.New(cfg, eal.NumaSocket{})
	if e != nil {
		panic(e)
	}
	fixture.Cs = cs.FromPcct(fixture.Pcct)
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

// Insert a CS entry, by replacing a PIT entry.
// Returns false if CS entry is found during PIT entry insertion.
// Returns true if CS entry is replacing PIT entry.
// This function takes ownership of interest and data.
func (fixture *Fixture) Insert(interest *ndni.Packet, data *ndni.Packet) (isReplacing bool) {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.FibEntry)
	if csEntry != nil {
		interest.Close()
		data.Close()
		return false
	}
	if pitEntry == nil {
		panic("Pit.Insert failed")
	}

	data.SetPitToken(pitEntry.PitToken())
	pitFound := fixture.Pit.FindByData(data)
	if len(pitFound.ListEntries()) == 0 {
		panic("Pit.FindByData returned empty result")
	}

	fixture.Cs.Insert(data, pitFound)
	return true
}

func (fixture *Fixture) InsertBulk(minId, maxId int, dataNameFmt, interestNameFmt string, makeInterestArgs ...interface{}) (nInserted int) {
	for i := minId; i <= maxId; i++ {
		dataName := fmt.Sprintf(dataNameFmt, i)
		interestName := fmt.Sprintf(interestNameFmt, i)
		interest := ndnitestenv.MakeInterest(interestName, makeInterestArgs...)
		data := ndnitestenv.MakeData(dataName, time.Second)
		ok := fixture.Insert(interest, data)
		if ok {
			nInserted++
		}
	}
	return nInserted
}

// Find a CS entry.
// If a PIT entry is created in Pit.Insert invocation, it is erased immediately.
// This function takes ownership of interest.
func (fixture *Fixture) Find(interest *ndni.Packet) *cs.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.FibEntry)
	if pitEntry != nil {
		fixture.Pit.Erase(pitEntry)
	} else {
		interest.Close()
	}
	return csEntry
}

func (fixture *Fixture) FindBulk(minId, maxId int, interestNameFmt string, makeInterestArgs ...interface{}) (nFound int) {
	for i := minId; i <= maxId; i++ {
		interestName := fmt.Sprintf(interestNameFmt, i)
		interest := ndnitestenv.MakeInterest(interestName, makeInterestArgs...)
		csEntry := fixture.Find(interest)
		if csEntry != nil {
			nFound++
		}
	}
	return nFound
}
