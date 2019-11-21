package fetch_test

import (
	"testing"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	var cfg fetch.FetcherConfig
	cfg.WindowCapacity = 1024

	fetcher, e := fetch.New(face, cfg)
	require.NoError(e)
	defer fetcher.Close()
	fetcher.SetLCore(pingtestenv.SlaveLCores[0])
	fetcher.SetName(ndn.MustParseName("/A"))

	rxQueue := pingtestenv.AttachRxQueue(fetcher)
	defer rxQueue.Close()

	nInterests := 0
	face.OnTxInterest(func(interest *ndn.Interest) {
		nInterests++
		data := ndntestutil.MakeData(interest.GetName().String())
		rxQueue.Rx(data)
	})

	fetcher.Logic.SetFinalSegNum(4999)
	fetcher.Launch()

	e = fetcher.WaitForCompletion()
	assert.NoError(e)
	assert.Equal(5000, nInterests)
}
