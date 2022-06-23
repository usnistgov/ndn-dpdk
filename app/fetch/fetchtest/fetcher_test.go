package fetchtest

import (
	"math/rand"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	defer intFace.D.Close()

	var cfg fetch.Config
	cfg.NThreads = 1
	cfg.NTasks = 2
	cfg.WindowCapacity = 1024

	fetcher, e := fetch.New(intFace.D, cfg)
	require.NoError(e)
	tgtestenv.Open(t, fetcher)
	defer fetcher.Close()
	fetcher.Launch()

	nInterests := map[byte]int{}
	go func() {
		for packet := range intFace.Rx {
			require.NotNil(packet.Interest)
			if assert.Len(packet.Lp.PitToken, 1) {
				nInterests[packet.Lp.PitToken[0]]++
			}
			if rand.Float64() > 0.01 {
				intFace.Tx <- ndn.MakeData(packet.Interest)
			}
		}
	}()

	var def0, def1 fetch.TaskDef
	def0.Prefix = ndn.ParseName("/A")
	def0.SegmentEnd = 5000
	task0, e := fetcher.Fetch(def0)
	require.NoError(e)
	def1.Prefix = ndn.ParseName("/B")
	def1.SegmentEnd = 2000
	task1, e := fetcher.Fetch(def1)
	require.NoError(e)

	t0 := time.Now()
	{
		ticker := time.NewTicker(time.Millisecond)
		for range ticker.C {
			if task0.Finished() && task1.Finished() {
				break
			}
		}
		ticker.Stop()
	}

	require.Len(nInterests, 2)
	assert.GreaterOrEqual(nInterests[0], 5000)
	assert.Less(nInterests[0], 6000)
	assert.GreaterOrEqual(nInterests[1], 2000)
	assert.Less(nInterests[1], 3000)
	t.Log(nInterests[0], "and", nInterests[1], "Interests in", time.Since(t0))
}
