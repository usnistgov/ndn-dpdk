package ndt_test

import (
	"math/rand"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/zyedidia/generic/mapset"
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
	started *sync.WaitGroup
	stop    ealthread.StopChan
	ndq     *ndt.Querier
	Entries []lookupTestEntry
	socket  eal.NumaSocket
}

func (th *lookupTestThread) NumaSocket() eal.NumaSocket {
	return th.socket
}

func (th *lookupTestThread) ThreadRole() string {
	return "TEST"
}

func (th *lookupTestThread) main() {
	th.started.Done()
	entries := make([]*lookupTestEntry, len(th.Entries))
	for i := range th.Entries {
		entries[i] = &th.Entries[i]
	}
	swapper := reflect.Swapper(entries)

	for th.stop.Continue() {
		rand.Shuffle(len(entries), swapper)
		for _, entry := range entries {
			result := th.ndq.Lookup(entry.Name)
			entry.AddResult(result)
		}
	}
}

func newLookupTestThread(t testing.TB, table *ndt.Ndt, names []ndn.Name, started *sync.WaitGroup) *lookupTestThread {
	ndq := eal.Zmalloc[ndt.Querier]("NdtQuerier", unsafe.Sizeof(ndt.Querier{}), eal.NumaSocket{})
	t.Cleanup(func() {
		eal.Free(ndq)
	})

	th := &lookupTestThread{
		started: started,
		stop:    ealthread.NewStopChan(),
		ndq:     ndq,
	}
	th.ndq.Init(table, eal.NumaSocket{})
	for _, name := range names {
		th.Entries = append(th.Entries, lookupTestEntry{name, nil})
	}
	th.Thread = ealthread.New(cptr.Func0.Void(th.main), th.stop)
	return th
}

func TestNdt(t *testing.T) {
	defer ealthread.AllocClear()
	assert, require := makeAR(t)

	cfg := ndt.Config{
		PrefixLen:      2,
		Capacity:       256,
		SampleInterval: 4,
	}
	table := ndt.New(cfg, nil)
	defer table.Close()

	var names []ndn.Name
	nameIndices := mapset.New[uint64]()
	for nameIndices.Size() != 7 {
		suffix := "_" + strconv.FormatUint(rand.Uint64(), 16)
		names = []ndn.Name{
			ndn.ParseName("/"),
			ndn.ParseName("/" + suffix),
			ndn.ParseName("/A" + suffix + "/2=C"),
			ndn.ParseName("/A" + suffix + "/A/C"),
			ndn.ParseName("/A" + suffix + "/A/D"),
			ndn.ParseName("/B" + suffix),
			ndn.ParseName("/B" + suffix + "/2=C"),
			ndn.ParseName("/B" + suffix + "/C"),
		}
		nameIndices.Clear()
		for _, name := range names {
			nameIndices.Put(table.IndexOfName(name))
		}
	}

	var started sync.WaitGroup
	started.Add(4)
	threads := []*lookupTestThread{
		newLookupTestThread(t, table, names[:6], &started),
		newLookupTestThread(t, table, names[:6], &started),
		newLookupTestThread(t, table, names[:6], &started),
		newLookupTestThread(t, table, names[6:], &started),
	}
	if len(eal.Sockets) >= 2 {
		threads[0].socket = eal.Sockets[0]
		threads[2].socket = eal.Sockets[0]
		threads[1].socket = eal.Sockets[1]
	}

	table.Randomize(250)
	list0 := table.List()
	for _, th := range threads {
		require.NoError(ealthread.AllocLaunch(th))
	}
	started.Wait()

	time.Sleep(400 * time.Millisecond)
	list1 := table.List()
	table.Randomize(250)
	time.Sleep(400 * time.Millisecond)

	for _, th := range threads {
		th.Stop()
	}
	time.Sleep(10 * time.Millisecond)
	list2 := table.List()

	// all counters are zero initially
	require.Len(list0, 256)
	for i, entry := range list0 {
		assert.EqualValues(i, entry.Index, i)
		assert.Zero(entry.Hits, i)
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
			assert.Equal(threads[0].Entries[i].Results, threads[j].Entries[i].Results, "%d %d", i, j)
		}
	}

	// /A/A/C and /A/A/D should have same results
	assert.Equal(threads[0].Entries[3].Results, threads[0].Entries[4].Results)

	verifyCnt := func(list []ndt.Entry) {
		require.Len(list, 256)
		for i, entry := range list {
			if nameIndices.Has(uint64(i)) {
				assert.NotZero(entry.Hits, i)
			} else {
				assert.Zero(entry.Hits, i)
			}
		}
	}
	verifyCnt(list1)
	verifyCnt(list2)
}
