package fetch_test

import (
	"fmt"
	"testing"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()

	var cfg fetch.FetcherConfig
	cfg.WindowCapacity = 1024
	fetcher, e := fetch.New(face, cfg)
	require.NoError(e)
	fetcher.SetLCore(pingtestenv.SlaveLCores[0])
	fetcher.SetName(ndn.MustParseName("/A"))

	rxQueue, e := dpdk.NewRing("FetcherRxQ", 1024, dpdk.NUMA_SOCKET_ANY, false, true)
	require.NoError(e)
	fetcher.SetRxQueue(rxQueue)

	face.OnTxInterest(func(interest *ndn.Interest) {
		fmt.Println(interest)
		data := ndntestutil.MakeData(interest.GetName().String())
		rxQueue.BurstEnqueue([]ndn.Packet{data.GetPacket()})
	})

	fetcher.Logic.SetFinalSegNum(4999)
	fetcher.Launch()

	e = fetcher.WaitForCompletion()
	assert.NoError(e)
	assert.Len(face.TxInterests, 5000)
}
