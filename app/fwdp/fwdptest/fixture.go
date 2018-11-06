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

	outputTxLoop *iface.MultiTxLoop
	faceIds      []iface.FaceId
}

func NewFixture(t *testing.T) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.require = require.New(t)

	var dpCfg fwdp.Config
	lcr := appinit.NewLCoreReservations()

	faceInputLc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
	fixture.require.True(faceInputLc.IsValid())
	dpCfg.InputLCores = []dpdk.LCore{faceInputLc}
	dpCfg.InputRxLoopers = []iface.IRxLooper{mockface.TheRxLoop}

	for i := 0; i < nFwds; i++ {
		lc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
		fixture.require.True(lc.IsValid())
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}

	outputLc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
	fixture.require.True(outputLc.IsValid())

	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023
	dpCfg.Crypto.OpPoolCacheSize = 31

	cryptoLc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
	fixture.require.True(cryptoLc.IsValid())
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

	fixture.outputTxLoop = iface.NewMultiTxLoop()
	outputLc.RemoteLaunch(func() int {
		fixture.outputTxLoop.TxLoop()
		return 0
	})

	e = theDp.Launch()
	fixture.require.NoError(e)

	return fixture
}

func (fixture *Fixture) Close() error {
	fixture.DataPlane.Stop()
	fixture.outputTxLoop.StopTxLoop()

	fixture.DataPlane.Close()
	iface.CloseAll()
	strategycode.CloseAll()
	return nil
}

func (fixture *Fixture) CreateFace() *mockface.MockFace {
	face, e := appinit.NewFaceFromUri(faceuri.MustParse("mock:"), nil)
	fixture.require.NoError(e)
	e = face.EnableThreadSafeTx(16)
	fixture.require.NoError(e)

	fixture.outputTxLoop.AddFace(face)
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
