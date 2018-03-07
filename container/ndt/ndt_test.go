package ndt_test

import (
	"sort"
	"testing"
	"time"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestNdt(t *testing.T) {
	assert, require := makeAR(t)
	slaves := dpdktestenv.Eal.Slaves[:8]

	cfg := ndt.Config{
		PrefixLen:  2,
		IndexBits:  8,
		SampleFreq: 2,
	}
	numaSockets := make([]dpdk.NumaSocket, len(slaves))
	for i, slave := range slaves {
		numaSockets[i] = slave.GetNumaSocket()
	}

	nameStrs := []string{
		"/",
		"/...",
		"/A/2=C",
		"/A/A/C",
		"/A/A/D",
		"/B",
		"/B/2=C",
		"/B/C",
	}
	names := make([]*ndn.Name, len(nameStrs))
	for i, nameStr := range nameStrs {
		var e error
		names[i], e = ndn.ParseName(nameStr)
		require.NoError(e, nameStr)
	}

	ndt := ndt.New(cfg, numaSockets)
	defer ndt.Close()

	ndt.Randomize(256)
	cnt0 := ndt.ReadCounters()

	const NLOOPS = 100000
	result1 := make([]uint8, len(slaves))
	result2 := make([]uint8, len(slaves))
	for i, slave := range slaves {
		ii := i
		ndtt := ndt.GetThread(i)
		name := names[i]
		slave.RemoteLaunch(func() int {
			for j := 0; j < NLOOPS; j++ {
				result := ndtt.Lookup(name)
				if result1[ii] == 0 || result1[ii] == result {
					result1[ii] = result // initial result
				} else if result2[ii] == 0 {
					result2[ii] = result // result after random update
				} else if result2[ii] != result {
					return j // shouldn't have third result
				}
			}
			return 0
		})
	}

	time.Sleep(10 * time.Millisecond)
	cnt1 := ndt.ReadCounters()
	ndt.Randomize(256)

	for i, slave := range slaves {
		assert.Zero(slave.Wait(), "%d", i)
		assert.NotZero(result1[i], "%d", i)
	}
	cnt2 := ndt.ReadCounters()

	for a := range names {
		for b := range names {
			if a >= b {
				continue
			}
			if a == 3 && b == 4 { // /A/A/C and /A/A/D have same 2-component prefix
				assert.Equal(result1[a], result1[b], "%d-%d", a, b)
				assert.Equal(result2[a], result2[b], "%d-%d", a, b)
			} else {
				assert.NotEqual(result1[a], result1[b], "%d-%d", a, b)
				assert.NotEqual(result2[a], result2[b], "%d-%d", a, b)
			}
		}
	}

	require.Len(cnt0, 256)
	sort.Ints(cnt0)
	assert.Zero(cnt0[255]) // all counters are zero initially

	require.Len(cnt1, 256)
	sort.Ints(cnt1)
	assert.Zero(cnt1[248])
	assert.NotZero(cnt1[249]) // seven counters are not zero, others are zero

	require.Len(cnt2, 256)
	sort.Ints(cnt2)
	unitCnt := NLOOPS >> uint(cfg.SampleFreq)
	assert.Equal([]int{0, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt * 2}, cnt2[248:])
}
