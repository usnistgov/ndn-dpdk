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

	fetcher, e := fetch.New(18, face, cfg)
	require.NoError(e)
	defer fetcher.Close()
	fetcher.SetLCore(pingtestenv.SlaveLCores[0])
	fetcher.SetName(ndn.MustParseName("/A"))

	rx := pingtestenv.MakeRxFunc(fetcher)
	nInterests := 0
	face.OnTxInterest(func(interest *ndn.Interest) {
		assert.EqualValues(ndntestutil.GetPitToken(interest)>>56, 18)
		nInterests++
		data := ndntestutil.MakeData(interest.GetName().String())
		rx(data)
	})

	fetcher.Logic.SetFinalSegNum(4999)
	fetcher.Launch()

	e = fetcher.WaitForCompletion()
	assert.NoError(e)
	assert.Equal(5000, nInterests)
}
