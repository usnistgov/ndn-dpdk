package cstest

import (
	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

type Fixture struct {
	Cs  cs.Cs
	Pit pit.Pit
}

func NewFixture(pcctMaxEntries int, csCapacity int) (fixture *Fixture) {
	cfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: pcctMaxEntries,
		CsCapacity: csCapacity,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}
	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}

	fixture = new(Fixture)
	fixture.Cs = cs.Cs{pcct}
	fixture.Pit = pit.Pit{pcct}
	return fixture
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
func (fixture *Fixture) Insert(interest *ndn.Interest, data *ndn.Data) bool {
	pitEntry, csEntry := fixture.Pit.Insert(interest)
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
	if pitFound.Len() == 0 {
		panic("Pit.FindByData returned empty result")
	}

	fixture.Cs.Insert(data, pitFound)
	return true
}

// Find a CS entry.
// If a PIT entry is created in Pit.Insert invocation, it is erased immediately.
// This function takes ownership of interest.
func (fixture *Fixture) Find(interest *ndn.Interest) *cs.Entry {
	pitEntry, csEntry := fixture.Pit.Insert(interest)
	if pitEntry != nil {
		fixture.Pit.Erase(*pitEntry)
	} else {
		ndntestutil.ClosePacket(interest)
	}
	return csEntry
}
