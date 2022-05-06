package fetchtest

import (
	"math/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	defer intFace.D.Close()

	cfg := fetch.FetcherConfig{
		NThreads:       1,
		NProcs:         1,
		WindowCapacity: 1024,
	}

	fetcher, e := fetch.New(intFace.D, cfg)
	require.NoError(e)
	tgtestenv.Open(t, fetcher)
	defer fetcher.Close()

	nInterests := 0
	go func() {
		for packet := range intFace.Rx {
			require.NotNil(packet.Interest)
			assert.Equal([]byte{0}, packet.Lp.PitToken)
			nInterests++
			if rand.Float64() > 0.01 {
				intFace.Tx <- ndn.MakeData(packet.Interest)
			}
		}
	}()

	fetcher.Reset()
	i, e := fetcher.AddTemplate(ndni.InterestTemplateConfig{
		Prefix: ndn.ParseName("/A"),
	})
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

	t.Log(nInterests, "Interests in", time.Since(t0))
	assert.GreaterOrEqual(nInterests, 5000)
	assert.Less(nInterests, 6000)
}
