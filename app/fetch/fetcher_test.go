package fetch_test

import (
	"testing"
	"time"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestenv"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	var cfg fetch.FetcherConfig
	cfg.NThreads = 1
	cfg.NProcs = 1
	cfg.WindowCapacity = 1024

	fetcher, e := fetch.New(face, cfg)
	require.NoError(e)
	defer fetcher.Close()
	fetcher.GetThread(0).SetLCore(pingtestenv.SlaveLCores[0])

	rx := pingtestenv.MakeRxFunc(fetcher.GetRxQueue(0))
	nInterests := 0
	face.OnTxInterest(func(interest *ndn.Interest) {
		assert.EqualValues(ndntestenv.GetPitToken(interest)>>56, 0)
		nInterests++
		data := ndntestenv.MakeData(interest.GetName().String())
		rx(data)
	})

	fetcher.Reset()
	i, e := fetcher.AddTemplate("/A")
	require.NoError(e)
	assert.Equal(i, 0)

	fetcher.GetLogic(i).SetFinalSegNum(4999)
	fetcher.Launch()

	time.Sleep(1500 * time.Millisecond)
	fetcher.Stop()
	assert.Equal(5000, nInterests)
}
