package fetch_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
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
	require.NoError(ealthread.DefaultAllocator.AllocThread(fetcher.Workers()...))
	fetcher.ConnectRxQueues(tgtestenv.DemuxD, tgtestenv.DemuxN, 0)

	nInterests := 0
	go func() {
		for packet := range intFace.Rx {
			require.NotNil(packet.Interest)
			token := ndn.PitTokenToUint(packet.Lp.PitToken)
			assert.NotZero(token)
			assert.EqualValues(0, token>>56)
			nInterests++
			if rand.Float32() > 0.01 {
				intFace.Tx <- ndn.MakeData(packet.Interest)
			}
		}
		close(intFace.Tx)
	}()

	fetcher.Reset()
	i, e := fetcher.AddTemplate("/A")
	require.NoError(e)
	assert.Equal(i, 0)

	logic := fetcher.Logic(i)
	logic.SetFinalSegNum(4999)
	fetcher.Launch()
	t0 := time.Now()

	{
		ticker := time.NewTicker(time.Millisecond)
		for range ticker.C {
			if logic.Finished() {
				break
			}
		}
		ticker.Stop()
	}
	fetcher.Stop()

	fmt.Println(nInterests, "Interests in", time.Since(t0))
	assert.GreaterOrEqual(nInterests, 5000)
	assert.Less(nInterests, 6000)
}
