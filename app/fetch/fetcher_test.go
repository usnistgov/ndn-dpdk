package fetch_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	face := makeMockFace()

	var cfg fetch.FetcherConfig
	cfg.WindowCapacity = 1024
	fetcher, e := fetch.New(face, cfg)
	require.NoError(e)
	fetcher.SetLCore(slaveLCores[0])
	fetcher.SetName(ndn.MustParseName("/A"))

	rxQueue, e := dpdk.NewRing("FetcherRxQ", 1024, dpdk.NUMA_SOCKET_ANY, false, true)
	require.NoError(e)
	fetcher.SetRxQueue(rxQueue)

	stop := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			for ; i < len(face.TxInterests); i++ {
				interest := face.TxInterests[i]
				for interest == nil { // workaround thread-unsafe append
					// TODO add a TX channel in MockFace for intercepting packets
					time.Sleep(time.Millisecond)
					interest = face.TxInterests[i]
				}
				fmt.Println(interest)
				data := ndntestutil.MakeData(interest.GetName().String())
				pkts := make([]ndn.Packet, 1)
				pkts[0] = data.GetPacket()
				rxQueue.BurstEnqueue(pkts)
			}
		}
	}()

	fetcher.Logic.SetFinalSegNum(4999)
	fetcher.Launch()

	// TODO add a "wait for completion" function
	time.Sleep(5 * time.Second)

	e = fetcher.Stop()
	stop <- true
	assert.NoError(e)
	assert.Len(face.TxInterests, 5000)
}
