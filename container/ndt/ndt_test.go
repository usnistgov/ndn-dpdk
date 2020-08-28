package ndt_test

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type lookupTestEntry struct {
	Name    ndn.Name
	Results []uint8
}

func (entry *lookupTestEntry) AddResult(result uint8) {
	if n := len(entry.Results); n == 0 || entry.Results[n-1] != result {
		entry.Results = append(entry.Results, result)
	}
}

type lookupTestThread struct {
	ealthread.Thread
	stop    ealthread.StopChan
	ndtt    *ndt.Thread
	Entries []lookupTestEntry
}

func newNdtLookupTestThread(ndtt *ndt.Thread, names []ndn.Name) *lookupTestThread {
	th := &lookupTestThread{
		stop: ealthread.NewStopChan(),
		ndtt: ndtt,
	}
	for _, name := range names {
		th.Entries = append(th.Entries, lookupTestEntry{name, nil})
	}
	th.Thread = ealthread.New(cptr.Func0.Void(th.main), th.stop)
	return th
}

func (th *lookupTestThread) ThreadRole() string {
	return "TEST"
}

func (th *lookupTestThread) main() {
	entries := make([]*lookupTestEntry, len(th.Entries))
	for i := range th.Entries {
		entries[i] = &th.Entries[i]
	}
	swapper := reflect.Swapper(entries)

	for th.stop.Continue() {
		rand.Shuffle(len(entries), swapper)
		for _, entry := range entries {
			result := th.ndtt.Lookup(entry.Name)
			entry.AddResult(result)
		}
	}
}

func TestNdt(t *testing.T) {
	defer ealthread.DefaultAllocator.Clear()
	assert, require := makeAR(t)

	cfg := ndt.Config{
		PrefixLen:      2,
		Capacity:       256,
		SampleInterval: 4,
	}
	table := ndt.New(cfg, make([]eal.NumaSocket, 4))
	defer table.Close()

	var names []ndn.Name
	var nameIndices map[uint64]bool
	for len(nameIndices) != 7 {
		suffix := "_" + strconv.FormatUint(rand.Uint64(), 16)
		nameUris := []string{
			"/",
			"/" + suffix,
			"/A" + suffix + "/2=C",
			"/A" + suffix + "/A/C",
			"/A" + suffix + "/A/D",
			"/B" + suffix,
			"/B" + suffix + "/2=C",
			"/B" + suffix + "/C",
		}
		names = make([]ndn.Name, len(nameUris))
		nameIndices = make(map[uint64]bool)
		for i, nameStr := range nameUris {
			names[i] = ndn.ParseName(nameStr)
			nameIndices[table.IndexOfName(names[i])] = true
		}
	}

	ndtts := table.Threads()
	threads := []*lookupTestThread{
		newNdtLookupTestThread(ndtts[0], names[:6]),
		newNdtLookupTestThread(ndtts[1], names[:6]),
		newNdtLookupTestThread(ndtts[2], names[:6]),
		newNdtLookupTestThread(ndtts[3], names[6:]),
	}

	table.Randomize(250)
	list0 := table.List()
	for _, th := range threads {
		require.NoError(ealthread.Launch(th))
	}

	time.Sleep(100 * time.Millisecond)
	list1 := table.List()
	table.Randomize(250)
	time.Sleep(100 * time.Millisecond)

	for _, th := range threads {
		th.Stop()
	}
	time.Sleep(10 * time.Millisecond)
	list2 := table.List()

	// all counters are zero initially
	require.Len(list0, 256)
	for i, entry := range list0 {
		assert.Equal(i, entry.Index, "%d", i)
		assert.Zero(entry.Hits, "%d", i)
	}

	// each name has one or two results
	for j, th := range threads {
		for i, entry := range th.Entries {
			nResults := len(entry.Results)
			assert.True(nResults == 1 || nResults == 2, "threads[%d].Entries[%d].Results=%v", j, i, entry.Results)
		}
	}

	// th0, th1, th2 should see consistent results
	for i := range names[:6] {
		for j := 1; j <= 2; j++ {
			assert.Equal(threads[0].Entries[i].Results, threads[j].Entries[i].Results)
		}
	}

	// /A/A/C and /A/A/D should have same results
	assert.Equal(threads[0].Entries[3].Results, threads[0].Entries[4].Results)

	verifyCnt := func(list []ndt.Entry) {
		require.Len(list, 256)
		for i, entry := range list {
			if nameIndices[uint64(i)] {
				assert.NotZero(entry.Hits, "%d", i)
			} else {
				assert.Zero(entry.Hits, "%d", i)
			}
		}
	}
	verifyCnt(list1)
	verifyCnt(list2)
}
