package fetchtest

import (
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	defer intFace.D.Close()

	var cfg fetch.Config
	cfg.NThreads = 1
	cfg.NTasks = 2
	cfg.WindowCapacity = 512

	fetcher, e := fetch.New(intFace.D, cfg)
	require.NoError(e)
	tgtestenv.Open(t, fetcher)
	defer fetcher.Close()
	fetcher.Launch()

	var defA, defB fetch.TaskDef
	defA.Prefix = ndn.ParseName("/A")
	defA.SegmentBegin, defA.SegmentEnd = 0, 5000
	defA.Filename, defA.SegmentLen = filepath.Join(t.TempDir(), "A.bin"), 100
	payloadA := make([]byte, int(defA.SegmentEnd)*defA.SegmentLen)
	randBytes(payloadA)
	defB.Prefix = ndn.ParseName("/B")
	defB.SegmentBegin, defB.SegmentEnd = 1000, 4000
	const finalBlockB = 1800

	pInterestsA, nInterestsA, pInterestsB, nInterestsB := map[tlv.NNI]int{}, 0, map[tlv.NNI]int{}, 0
	go func() {
		for packet := range intFace.Rx {
			require.NotNil(packet.Interest)
			data := ndn.MakeData(packet.Interest, time.Millisecond)

			lastComp := packet.Interest.Name.Get(-1)
			assert.EqualValues(an.TtSegmentNameComponent, lastComp.Type)
			var segNum tlv.NNI
			assert.NoError(segNum.UnmarshalBinary(lastComp.Value))

			switch {
			case defA.Prefix.IsPrefixOf(packet.Interest.Name):
				nInterestsA++
				pInterestsA[segNum]++
				assert.Less(uint64(segNum), defA.SegmentEnd)
				payloadOffset := int(segNum) * defA.SegmentLen
				data.Content = payloadA[payloadOffset : payloadOffset+defA.SegmentLen]
			case defB.Prefix.IsPrefixOf(packet.Interest.Name):
				nInterestsB++
				pInterestsB[segNum]++
				assert.GreaterOrEqual(uint64(segNum), defB.SegmentBegin)
				assert.Less(uint64(segNum), defB.SegmentEnd)
				if segNum == finalBlockB {
					data.FinalBlock = lastComp
				} else if segNum > finalBlockB {
					continue
				}
			default:
				assert.Fail("unexpected Interest", packet.Interest.Name)
			}

			if rand.Float64() > 0.01 {
				intFace.Tx <- data
			}
		}
	}()

	taskA, e := fetcher.Fetch(defA)
	require.NoError(e)
	taskB, e := fetcher.Fetch(defB)
	require.NoError(e)

	t0 := time.Now()
	{
		ticker := time.NewTicker(time.Millisecond)
		for range ticker.C {
			if taskA.Finished() && taskB.Finished() {
				break
			}
		}
		ticker.Stop()
	}

	cntA, cntB := taskA.Counters(), taskB.Counters()
	assert.EqualValues(defA.SegmentEnd-defA.SegmentBegin, cntA.NRxData)
	assert.EqualValues(defA.SegmentEnd-defA.SegmentBegin, len(pInterestsA))
	assert.Zero(cntA.NInFlight)
	assert.InDelta(float64(nInterestsA), float64(cntA.NTxRetx+cntA.NRxData), float64(cfg.WindowCapacity))

	assert.EqualValues(finalBlockB-defB.SegmentBegin+1, cntB.NRxData)
	assert.GreaterOrEqual(len(pInterestsB), int(finalBlockB-defB.SegmentBegin+1))
	assert.Less(len(pInterestsB), int(finalBlockB-defB.SegmentBegin)+cfg.WindowCapacity)

	t.Logf("/A Interests %d (unique %d) and /B Interests %d (unique %d) in %v",
		nInterestsA, len(pInterestsA), nInterestsB, len(pInterestsB), time.Since(t0))

	if fA, e := os.Open(defA.Filename); assert.NoError(e) {
		defer fA.Close()
		writtenA, e := io.ReadAll(fA)
		assert.NoError(e)
		assert.Equal(payloadA, writtenA)
	}
}
