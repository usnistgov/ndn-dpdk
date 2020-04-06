package cs_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestMain(m *testing.M) {
	// fixture.Close() cannot release packet buffers, need a large mempool
	dpdktestenv.MakeDirectMp(65535, ndn.SizeofPacketPriv(), 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

type Fixture struct {
	Cs            cs.Cs
	Pit           pit.Pit
	emptyFibEntry *fib.Entry
}

func NewFixture(cfg pcct.Config) (fixture *Fixture) {
	cfg.Id = "TestPcct"
	cfg.MaxEntries = 4095
	cfg.NumaSocket = dpdk.NUMA_SOCKET_ANY
	if cfg.CsCapMd == 0 {
		cfg.CsCapMd = 200
	}
	if cfg.CsCapMi == 0 {
		cfg.CsCapMd = 200
	}

	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}

	return &Fixture{
		Cs:            cs.Cs{pcct},
		Pit:           pit.Pit{pcct},
		emptyFibEntry: new(fib.Entry),
	}
}

func (fixture *Fixture) Close() error {
	return fixture.Cs.Pcct.Close()
}

// Return number of in-use entries in PCCT's underlying mempool.
func (fixture *Fixture) CountMpInUse() int {
	return fixture.Cs.GetMempool().CountInUse()
}

// Insert a CS entry, by replacing a PIT entry.
// Returns false if CS entry is found during PIT entry insertion.
// Returns true if CS entry is replacing PIT entry.
// This function takes ownership of interest and data.
func (fixture *Fixture) Insert(interest *ndn.Interest, data *ndn.Data) (isReplacing bool) {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.emptyFibEntry)
	if csEntry != nil {
		ndntestutil.ClosePacket(interest)
		ndntestutil.ClosePacket(data)
		return false
	}
	if pitEntry == nil {
		panic("Pit.Insert failed")
	}

	ndntestutil.SetPitToken(data, pitEntry.GetToken())
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
		interest := ndntestutil.MakeInterest(interestName, makeInterestArgs...)
		data := ndntestutil.MakeData(dataName, time.Second)
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
func (fixture *Fixture) Find(interest *ndn.Interest) *cs.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest, fixture.emptyFibEntry)
	if pitEntry != nil {
		fixture.Pit.Erase(*pitEntry)
	} else {
		ndntestutil.ClosePacket(interest)
	}
	return csEntry
}

func (fixture *Fixture) FindBulk(minId, maxId int, interestNameFmt string, makeInterestArgs ...interface{}) (nFound int) {
	for i := minId; i <= maxId; i++ {
		interestName := fmt.Sprintf(interestNameFmt, i)
		interest := ndntestutil.MakeInterest(interestName, makeInterestArgs...)
		csEntry := fixture.Find(interest)
		if csEntry != nil {
			nFound++
		}
	}
	return nFound
}
