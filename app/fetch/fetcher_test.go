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
	defer intFace.DFace.Close()

	var cfg fetch.FetcherConfig
	cfg.NThreads = 1
	cfg.NProcs = 1
	cfg.WindowCapacity = 1024

	fetcher, e := fetch.New(intFace.DFace, cfg)
	require.NoError(e)
	defer fetcher.Close()
	fetcher.GetThread(0).SetLCore(pingtestenv.SlaveLCores[0])
	pingtestenv.Demux3.GetDataDemux().SetDest(0, fetcher.GetRxQueue(0))

	nInterests := 0
	go func() {
		tx := intFace.AFace.GetTx()
		for packet := range intFace.AFace.GetRx() {
			require.NotNil(packet.Interest)
			token := ndn.PitTokenToUint(packet.Lp.PitToken)
			assert.NotZero(token)
			assert.EqualValues(0, token>>56)
			nInterests++
			tx <- ndn.MakeData(*packet.Interest).Packet
		}
		close(tx)
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
