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

const testFetcherWindowCapacity = 512

type testFetcherTask struct {
	fetch.TaskDef
	FinalBlock int64
	Payload    []byte
	PInterests map[tlv.NNI]int
	NInterests int
}

func (ft *testFetcherTask) Run(t *testing.T, fetcher *fetch.Fetcher) (cnt fetch.Counters) {
	t.Parallel()
	assert, require := makeAR(t)

	task, e := fetcher.Fetch(ft.TaskDef)
	require.NoError(e)

	t0 := time.Now()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if task.Finished() {
			break
		}
	}
	task.Stop()

	cnt = task.Counters()
	t.Logf("Interests %d (unique %d) in %v", ft.NInterests, len(ft.PInterests), time.Since(t0))
	t.Logf("Counters %v", cnt)

	if ft.Filename != "" {
		if fd, e := os.Open(ft.Filename); assert.NoError(e) {
			defer fd.Close()
			written, e := io.ReadAll(fd)
			assert.NoError(e)
			assert.Equal(ft.Payload, written)
		}
	}

	return
}

func (ft *testFetcherTask) Serve(segNum tlv.NNI, lastComp ndn.NameComponent, data *ndn.Data) bool {
	ft.NInterests++
	ft.PInterests[segNum]++

	if uint64(segNum) < ft.SegmentBegin || (ft.SegmentEnd > 0 && uint64(segNum) >= ft.SegmentEnd) {
		panic(segNum)
	}

	if ft.Payload != nil {
		payloadOffset := int(segNum) * ft.SegmentLen
		data.Content = ft.Payload[payloadOffset:min(int(payloadOffset+ft.SegmentLen), len(ft.Payload))]
	}
	if ft.FinalBlock >= 0 {
		if int64(segNum) == ft.FinalBlock {
			data.FinalBlock = lastComp
		} else if int64(segNum) > ft.FinalBlock {
			return false
		}
	}
	return true
}

func newTestFetcherTask(prefix rune) *testFetcherTask {
	var td testFetcherTask
	td.Prefix = ndn.ParseName("/" + string(prefix))
	td.FinalBlock = -1
	td.PInterests = map[tlv.NNI]int{}
	return &td
}

func TestFetcher(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	t.Cleanup(func() { intFace.D.Close() })

	var cfg fetch.Config
	cfg.NThreads = 2
	cfg.NTasks = 8
	cfg.WindowCapacity = testFetcherWindowCapacity

	fetcher, e := fetch.New(intFace.D, cfg)
	require.NoError(e)
	tgtestenv.Open(t, fetcher)
	t.Cleanup(func() { fetcher.Close() })
	fetcher.Launch()

	tempDir := t.TempDir()
	ftByName := map[rune]*testFetcherTask{}

	t.Run("0", func(t *testing.T) {
		assert, _ := makeAR(t)

		ft := newTestFetcherTask('0')
		ft.SegmentBegin, ft.SegmentEnd = 1000, 1000
		ftByName['0'] = ft
		// empty SegmentRange

		cnt := ft.Run(t, fetcher)
		assert.Zero(cnt.NRxData)
		assert.Zero(len(ft.PInterests))
	})

	t.Run("A", func(t *testing.T) {
		assert, _ := makeAR(t)

		ft := newTestFetcherTask('A')
		ft.SegmentBegin, ft.SegmentEnd = 1000, 4000
		ft.FinalBlock = 1800
		ftByName['A'] = ft
		// bounded by both SegmentRange and FinalBlock

		cnt := ft.Run(t, fetcher)
		assert.EqualValues(ft.FinalBlock-int64(ft.SegmentBegin)+1, cnt.NRxData)
		nUniqueInterests := int64(len(ft.PInterests))
		assert.GreaterOrEqual(nUniqueInterests, ft.FinalBlock-int64(ft.SegmentBegin)+1)
		assert.Less(nUniqueInterests, ft.FinalBlock-int64(ft.SegmentBegin)+testFetcherWindowCapacity)
	})

	t.Run("H", func(t *testing.T) {
		assert, _ := makeAR(t)

		ft := newTestFetcherTask('H')
		ft.SegmentBegin, ft.SegmentEnd, ft.SegmentLen = 0, 5000, 100
		ft.Filename = filepath.Join(tempDir, "H.bin")
		ft.Payload = make([]byte, int64(ft.SegmentLen)*int64(ft.SegmentEnd))
		randBytes(ft.Payload)
		ftByName['H'] = ft
		// bounded by SegmentRange, write to file

		cnt := ft.Run(t, fetcher)
		assert.EqualValues(ft.SegmentEnd-ft.SegmentBegin, cnt.NRxData)
		assert.EqualValues(ft.SegmentEnd-ft.SegmentBegin, len(ft.PInterests))
		assert.Zero(cnt.NInFlight)
		assert.InDelta(float64(ft.NInterests), float64(cnt.NTxRetx+cnt.NRxData), testFetcherWindowCapacity)
	})

	t.Run("I", func(t *testing.T) {
		assert, _ := makeAR(t)

		ft := newTestFetcherTask('I')
		ft.SegmentBegin, ft.SegmentEnd, ft.SegmentLen = 0, 900, 400
		ft.Filename = filepath.Join(tempDir, "I.bin")
		fileSize := int64(ft.SegmentLen)*int64(ft.SegmentEnd) - 7
		ft.FileSize = &fileSize
		ft.Payload = make([]byte, fileSize)
		randBytes(ft.Payload)
		ftByName['I'] = ft
		// bounded by SegmentRange, write to file, truncate file

		cnt := ft.Run(t, fetcher)
		assert.EqualValues(ft.SegmentEnd-ft.SegmentBegin, cnt.NRxData)
		assert.EqualValues(ft.SegmentEnd-ft.SegmentBegin, len(ft.PInterests))
		assert.Zero(cnt.NInFlight)
		assert.InDelta(float64(ft.NInterests), float64(cnt.NTxRetx+cnt.NRxData), testFetcherWindowCapacity)
	})

	go func() {
		for packet := range intFace.Rx {
			if !assert.NotNil(packet.Interest) || !assert.Len(packet.Interest.Name, 2) {
				continue
			}
			data := ndn.MakeData(packet.Interest, time.Millisecond)

			comp0, comp1 := packet.Interest.Name[0], packet.Interest.Name[1]
			assert.EqualValues(an.TtGenericNameComponent, comp0.Type)
			assert.EqualValues(1, comp0.Length())
			assert.EqualValues(an.TtSegmentNameComponent, comp1.Type)
			var segNum tlv.NNI
			assert.NoError(segNum.UnmarshalBinary(comp1.Value))

			respond := false
			ft := ftByName[rune(comp0.Value[0])]
			if ft == nil {
				assert.Fail("unexpected Interest", packet.Interest.Name)
			} else {
				respond = ft.Serve(segNum, comp1, &data)
			}
			if respond && rand.Float64() > 0.01 {
				intFace.Tx <- data
			}
		}
	}()
}
