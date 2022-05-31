package tgconsumer

import (
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func formatRttCounters(cnt runningstat.Snapshot) string {
	const ratio = 1.0 / float64(time.Millisecond)
	formatMinMax := func(v *uint64) string {
		if v == nil {
			return "none"
		}
		return strconv.FormatFloat(float64(*v)*ratio, 'f', 3, 64)
	}
	ms := cnt.Scale(ratio)
	return fmt.Sprintf("%s/%0.3f/%s/%0.3fms(%dsamp)", formatMinMax(ms.Min), ms.Mean, formatMinMax(ms.Max), ms.Stdev, ms.Len)
}

// PacketCounters is a group of network layer packet counters.
type PacketCounters struct {
	NInterests uint64 `json:"nInterests"`
	NData      uint64 `json:"nData"`
	NNacks     uint64 `json:"nNacks"`
}

// DataRatio returns NData/NInterests.
func (cnt PacketCounters) DataRatio() float64 {
	return float64(cnt.NData) / float64(cnt.NInterests)
}

// NackRatio returns NNacks/NInterests.
func (cnt PacketCounters) NackRatio() float64 {
	return float64(cnt.NNacks) / float64(cnt.NInterests)
}

func (cnt PacketCounters) String() string {
	return fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%)",
		cnt.NInterests,
		cnt.NData, cnt.DataRatio()*100.0,
		cnt.NNacks, cnt.NackRatio()*100.0)
}

// PatternCounters contains per-pattern counters.
type PatternCounters struct {
	PacketCounters
	Rtt runningstat.Snapshot `json:"rtt" gqldesc:"RTT in nanoseconds."`
}

func (cnt PatternCounters) String() string {
	return fmt.Sprintf("%s rtt=%s", cnt.PacketCounters, formatRttCounters(cnt.Rtt))
}

// Counters contains consumer counters.
type Counters struct {
	PacketCounters
	NAllocError uint64               `json:"nAllocError"`
	Rtt         runningstat.Snapshot `json:"rtt" gqldesc:"RTT in nanoseconds."`
	PerPattern  []PatternCounters    `json:"perPattern"`
}

func (cnt Counters) String() string {
	s := fmt.Sprintf("%s %dalloc-error rtt=%s", cnt.PacketCounters, cnt.NAllocError, formatRttCounters(cnt.Rtt))
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

// Counters retrieves counters.
func (c *Consumer) Counters() (cnt Counters) {
	for i := 0; i < int(c.rxC.nPatterns); i++ {
		crP := c.rxC.pattern[i]
		ctP := c.txC.pattern[i]
		rtt := c.rttStat(i).Read().Scale(eal.TscNanos)

		var pcnt PatternCounters
		pcnt.NInterests = uint64(ctP.nInterests)
		pcnt.NData = rtt.Count
		pcnt.NNacks = uint64(crP.nNacks)
		pcnt.Rtt = rtt
		cnt.PerPattern = append(cnt.PerPattern, pcnt)

		cnt.NInterests += pcnt.NInterests
		cnt.NData += pcnt.NData
		cnt.NNacks += pcnt.NNacks
		cnt.Rtt = cnt.Rtt.Add(rtt)
	}

	cnt.NAllocError = uint64(c.txC.nAllocError)
	return cnt
}

func (c *Consumer) rttStat(index int) *runningstat.IntStat {
	return runningstat.IntFromPtr(unsafe.Pointer(&c.rxC.pattern[index].rtt))
}

// ClearCounters clears counters.
// Both RX and TX threads should be stopped before calling this, otherwise race conditions may occur.
func (c *Consumer) ClearCounters() {
	nPatterns := int(c.rxC.nPatterns)
	for i := 0; i < nPatterns; i++ {
		c.clearCounter(i)
	}
	c.txC.nAllocError = 0
}

func (c *Consumer) clearCounter(index int) {
	c.rxC.pattern[index].nNacks = 0
	c.rttStat(index).Init(0)
	c.txC.pattern[index].nInterests = 0
}
