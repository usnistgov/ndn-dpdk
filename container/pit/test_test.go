package pit_test

import (
	"os"
	"testing"

	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

type Fixture struct {
	Pit pit.Pit
}

func NewFixture(pcctMaxEntries int) (fixture *Fixture) {
	cfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: pcctMaxEntries,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}
	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}

	fixture = new(Fixture)
	fixture.Pit = pit.Pit{pcct}
	return fixture
}

func (fixture *Fixture) Close() error {
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
	pitEntry, csEntry := fixture.Pit.Insert(interest)
	if csEntry != nil {
		ndntestutil.ClosePacket(interest)
		return nil
	}
	if pitEntry == nil {
		panic("Pit.Insert failed")
	}
	return pitEntry
}
