package ndt_test

import (
	"math/rand"
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
		"0700",
		"0702 0800",
		"0706 080141 020143",
		"0709 080141 080141 080143",
		"0709 080141 080141 080144",
		"0703 080142",
		"0706 080142 020143",
		"0706 080142 080143",
	}
	namePkts := make([]dpdk.Packet, len(nameStrs))
	names := make([]*ndn.Name, len(nameStrs))
	for i, nameStr := range nameStrs {
		namePkts[i] = dpdktestenv.PacketFromHex(nameStr)
		defer namePkts[i].Close()
		d := ndn.NewTlvDecoder(namePkts[i])
		name, e := d.ReadName()
		require.NoError(e)
		names[i] = &name
	}
	require.Equal(len(names), len(slaves))

	ndt := ndt.New(cfg, numaSockets)
	defer ndt.Close()

	randomUpdate := func() {
		for h := 0; h < (1 << uint(cfg.IndexBits)); h++ {
			v := uint8(0)
			for v == 0 {
				v = uint8(rand.Int())
			}
			ndt.Update(uint64(h), v)
		}
	}
	randomUpdate()
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
					result2[ii] = result // result from random update
				} else if result2[ii] != result {
					return j
				}
			}
			return 0
		})
	}

	time.Sleep(10 * time.Millisecond)
	cnt1 := ndt.ReadCounters()
	randomUpdate()

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
			if a == 3 && b == 4 { // they have same 2-component prefix
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
	assert.Zero(cnt0[255])

	require.Len(cnt1, 256)
	sort.Ints(cnt1)
	assert.Zero(cnt1[248])
	assert.NotZero(cnt1[249])

	require.Len(cnt2, 256)
	sort.Ints(cnt2)
	unitCnt := NLOOPS >> uint(cfg.SampleFreq)
	assert.Equal([]int{0, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt, unitCnt * 2}, cnt2[248:])
}
