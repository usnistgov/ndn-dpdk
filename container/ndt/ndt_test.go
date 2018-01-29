package ndt_test

import (
	"math/rand"
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
		SampleFreq: 1,
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
	randomUpdate()

	for i, slave := range slaves {
		assert.Zero(slave.Wait(), "%d", i)
		assert.NotZero(result1[i], "%d", i)
	}

	equalPairs := map[[2]int]bool{
		{3, 4}: true,
	}
	for a := range names {
		for b := range names {
			if a >= b {
				continue
			}
			if equalPairs[[2]int{a, b}] {
				assert.Equal(result1[a], result1[b], "%d-%d", a, b)
				assert.Equal(result2[a], result2[b], "%d-%d", a, b)
			} else {
				assert.NotEqual(result1[a], result1[b], "%d-%d", a, b)
				assert.NotEqual(result2[a], result2[b], "%d-%d", a, b)
			}
		}
	}
}
