package fetch_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/ping/pingtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	defer intFace.D.Close()

	var cfg fetch.FetcherConfig
	cfg.NThreads = 1
	cfg.NProcs = 1
	cfg.WindowCapacity = 1024

	fetcher, e := fetch.New(intFace.D, cfg)
	require.NoError(e)
	defer fetcher.Close()
	fetcher.GetThread(0).SetLCore(pingtestenv.WorkerLCores[0])
	pingtestenv.Demux3.GetDataDemux().SetDest(0, fetcher.GetRxQueue(0))

	nInterests := 0
	go func() {
		for packet := range intFace.Rx {
			require.NotNil(packet.Interest)
			token := ndn.PitTokenToUint(packet.Lp.PitToken)
			assert.NotZero(token)
			assert.EqualValues(0, token>>56)
			nInterests++
			intFace.Tx <- ndn.MakeData(packet.Interest)
		}
		close(intFace.Tx)
	}()

	fetcher.Reset()
	i, e := fetcher.AddTemplate("/A")
	require.NoError(e)
	assert.Equal(i, 0)

	fetcher.GetLogic(i).SetFinalSegNum(4999)
	fetcher.Launch()

	time.Sleep(1500 * time.Millisecond)
	fetcher.Stop()
	assert.GreaterOrEqual(nInterests, 4000)
}
