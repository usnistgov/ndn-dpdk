package fwdptestfixture

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
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

	Ndt       ndt.Ndt
	Fib       *fib.Fib
	DataPlane *fwdp.DataPlane

	outputTxLoop *iface.MultiTxLoop
	faceIds      []iface.FaceId
	strategies   map[string]fib.StrategyCode
}

func New(t *testing.T) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.require = require.New(t)

	var dpCfg fwdp.Config
	lcr := appinit.NewLCoreReservations()

	inputLc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
	fixture.require.True(inputLc.IsValid())
	dpCfg.InputLCores = []dpdk.LCore{inputLc}

	for i := 0; i < nFwds; i++ {
		lc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
		fixture.require.True(lc.IsValid())
		dpCfg.FwdLCores = append(dpCfg.FwdLCores, lc)
	}

	outputLc := lcr.Reserve(dpdk.NUMA_SOCKET_ANY)
	fixture.require.True(outputLc.IsValid())

	{
		var ndtCfg ndt.Config
		ndtCfg.PrefixLen = 2
		ndtCfg.IndexBits = 16
		ndtCfg.SampleFreq = 8
		theNdt := ndt.New(ndtCfg, dpdk.ListNumaSocketsOfLCores(dpCfg.InputLCores))
		fixture.Ndt = theNdt
		dpCfg.Ndt = theNdt
		theNdt.Randomize(nFwds)
	}

	{
		var fibCfg fib.Config
		fibCfg.Id = "FIB"
		fibCfg.MaxEntries = 65535
		fibCfg.NBuckets = 256
		fibCfg.StartDepth = 8
		theFib, e := fib.New(fibCfg, fixture.Ndt, dpdk.ListNumaSocketsOfLCores(dpCfg.FwdLCores))
		fixture.require.NoError(e)
		fixture.Fib = theFib
		dpCfg.Fib = theFib
	}

	dpCfg.FwdQueueCapacity = 64
	dpCfg.PcctCfg.MaxEntries = 65535

	theDp, e := fwdp.New(dpCfg)
	fixture.require.NoError(e)
	fixture.DataPlane = theDp

	e = fixture.DataPlane.LaunchInput(0, mockface.TheRxLoop, 1)
	fixture.require.NoError(e)
	for i := 0; i < nFwds; i++ {
		e := fixture.DataPlane.LaunchFwd(i)
		fixture.require.NoError(e)
	}
	fixture.outputTxLoop = iface.NewMultiTxLoop()
	outputLc.RemoteLaunch(func() int {
		fixture.outputTxLoop.TxLoop()
		return 0
	})

	fixture.strategies = make(map[string]fib.StrategyCode)
	return fixture
}

func (fixture *Fixture) Close() error {
	fixture.DataPlane.StopInput(0)
	for i := 0; i < nFwds; i++ {
		fixture.DataPlane.StopFwd(i)
	}
	fixture.outputTxLoop.StopTxLoop()

	fixture.DataPlane.Close()
	fixture.Ndt.Close()
	fixture.Fib.Close()
	iface.CloseAll()
	return nil
}

func (fixture *Fixture) CreateFace() *mockface.MockFace {
	face, e := appinit.NewFaceFromUri(faceuri.MustParse("mock://"), nil)
	fixture.require.NoError(e)
	e = face.EnableThreadSafeTx(16)
	fixture.require.NoError(e)

	fixture.outputTxLoop.AddFace(face)
	faceId := face.GetFaceId()
	fixture.faceIds = append(fixture.faceIds, faceId)
	return face.(*mockface.MockFace)
}

func (fixture *Fixture) SetFibEntry(name string, strategy string, nexthops ...iface.FaceId) {
	n, e := ndn.ParseName(name)
	fixture.require.NoError(e)

	var entry fib.Entry
	e = entry.SetName(n)
	fixture.require.NoError(e)

	e = entry.SetNexthops(nexthops)
	fixture.require.NoError(e)

	entry.SetStrategy(fixture.makeStrategy(strategy))

	_, e = fixture.Fib.Insert(&entry)
	fixture.require.NoError(e)
}

func (fixture *Fixture) makeStrategy(shortname string) fib.StrategyCode {
	if sc, ok := fixture.strategies[shortname]; ok {
		return sc
	}

	elf, e := strategy_elf.Load(shortname)
	fixture.require.NoError(e)

	sc, e := fixture.Fib.LoadStrategyCode(elf)
	fixture.require.NoError(e)

	fixture.strategies[shortname] = sc
	return sc
}
