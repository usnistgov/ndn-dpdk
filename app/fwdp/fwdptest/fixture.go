package fwdptest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/strategy/strategy_elf"
)

const nFwds = 2

type Fixture struct {
	require *require.Assertions

	DataPlane *fwdp.DataPlane
	Ndt       *ndt.Ndt
	Fib       *fib.Fib

	faceIds []iface.FaceId
}

func NewFixture(t *testing.T) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.require = require.New(t)

	var dpCfg fwdp.Config
	lcr := appinit.NewLCoreReservations()

	for i := 0; i < nFwds; i++ {
		lc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}

	faceInputLc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
	dpCfg.InputLCores = append(dpCfg.InputLCores, faceInputLc)

	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023
	dpCfg.Crypto.OpPoolCacheSize = 31
	cryptoLc := lcr.MustReserve(dpdk.NUMA_SOCKET_ANY)
	dpCfg.CryptoLCore = cryptoLc

	dpCfg.Ndt.PrefixLen = 2
	dpCfg.Ndt.IndexBits = 16
	dpCfg.Ndt.SampleFreq = 8

	dpCfg.Fib.MaxEntries = 65535
	dpCfg.Fib.NBuckets = 256
	dpCfg.Fib.StartDepth = 8

	dpCfg.Pcct.MaxEntries = 65535
	dpCfg.Pcct.CsCapacity = 32767

	dpCfg.FwdQueueCapacity = 64
	dpCfg.LatencySampleFreq = 0

	theDp, e := fwdp.New(dpCfg)
	fixture.require.NoError(e)
	fixture.DataPlane = theDp
	fixture.Ndt = theDp.GetNdt()
	fixture.Fib = theDp.GetFib()

	e = theDp.Launch()
	fixture.require.NoError(e)

	appinit.TxlLCoreReservation = lcr
	var faceCfg createface.Config
	faceCfg.EnableMock = true
	faceCfg.MockTxqPkts = 16
	appinit.EnableCreateFace(faceCfg) // ignore double-init error

	return fixture
}

func (fixture *Fixture) Close() error {
	iface.CloseAll()
	fixture.DataPlane.Stop()
	fixture.DataPlane.Close()
	strategycode.CloseAll()
	return nil
}

func (fixture *Fixture) CreateFace() *mockface.MockFace {
	var faceArg createface.CreateArg
	faceArg.Remote = faceuri.MustParse("mock:")
	faces, e := createface.Create(faceArg)
	fixture.require.NoError(e)
	fixture.require.Len(faces, 1)

	face := faces[0]
	faceId := face.GetFaceId()
	fixture.faceIds = append(fixture.faceIds, faceId)
	return face.(*mockface.MockFace)
}

func (fixture *Fixture) SetFibEntry(name string, strategy string, nexthops ...iface.FaceId) {
	var entry fib.Entry
	e := entry.SetName(ndn.MustParseName(name))
	fixture.require.NoError(e)

	e = entry.SetNexthops(nexthops)
	fixture.require.NoError(e)

	entry.SetStrategy(fixture.makeStrategy(strategy))

	_, e = fixture.Fib.Insert(&entry)
	fixture.require.NoError(e)
}

func (fixture *Fixture) ReadFibCounters(name string) fib.EntryCounters {
	return fixture.Fib.ReadEntryCounters(ndn.MustParseName(name))
}

func (fixture *Fixture) makeStrategy(shortname string) strategycode.StrategyCode {
	if sc, ok := strategycode.Find(shortname); ok {
		return sc
	}

	elf, e := strategy_elf.Load(shortname)
	fixture.require.NoError(e)

	sc, e := strategycode.Load(shortname, elf)
	fixture.require.NoError(e)

	return sc
}

// Read a counter from all FwFwds and compute the sum.
func (fixture *Fixture) SumCounter(getCounter func(dp *fwdp.DataPlane, i int) uint64) (n uint64) {
	for i := 0; i < nFwds; i++ {
		n += getCounter(fixture.DataPlane, i)
	}
	return n
}
