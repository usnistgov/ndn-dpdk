package ethface_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/ndn"
)

func TestEthFace(t *testing.T) {
	assert, require := dpdktestenv.MakeAR(t)

	dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 2000)
	mempools := iface.Mempools{
		IndirectMp: dpdktestenv.MakeIndirectMp(4095),
		NameMp:     dpdktestenv.MakeMp("name", 4095, 0, ndn.NAME_MAX_LENGTH),
		HeaderMp:   dpdktestenv.MakeMp("header", 4095, 0, ethface.SizeofTxHeader()),
	}
	evl := dpdktestenv.NewEthVLink(2, 1024, 64, dpdktestenv.MPID_DIRECT)
	defer evl.Close()

	faceA, e := ethface.New(evl.PortA, mempools)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := ethface.New(evl.PortB, mempools)
	require.NoError(e)
	defer faceB.Close()

	rxlA := ethface.NewRxLoop(3, faceA.GetNumaSocket())
	defer rxlA.Close()
	require.NoError(rxlA.Add(faceA)) // queue 0
	require.NoError(rxlA.Add(faceA)) // queue 1
	assert.Error(rxlA.Add(faceA))    // queue 2 does not exist

	fixture := ifacetestfixture.New(t, faceA, rxlA, faceB)
	dpdktestenv.Eal.Slaves[2].RemoteLaunch(evl.Bridge)
	time.Sleep(time.Second)
	fixture.RunTest()
	fixture.CheckCounters()

	fmt.Println("TX port", evl.PortB.GetStats())
	fmt.Println("TX face", faceB.ReadCounters())
	fmt.Println("RX port", evl.PortA.GetStats())
	fmt.Println("RX face", faceA.ReadCounters())
	fmt.Println("AtoB", evl.AtoB)
	fmt.Println("BtoA", evl.BtoA)
}
