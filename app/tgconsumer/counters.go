package tgconsumer

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

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

// RttCounters contains RTT statistics in nanoseconds.
type RttCounters struct {
	runningstat.Snapshot
}

func (cnt RttCounters) String() string {
	ms := cnt.Scale(1.0 / float64(time.Millisecond))
	return fmt.Sprintf("%0.3f/%0.3f/%0.3f/%0.3fms(%dsamp)", ms.Min(), ms.Mean(), ms.Max(), ms.Stdev(), ms.Len())
}

// PatternCounters contains per-pattern counters.
type PatternCounters struct {
	PacketCounters
	Rtt RttCounters `json:"rtt"`
}

func (cnt PatternCounters) String() string {
	return fmt.Sprintf("%s rtt=%s", cnt.PacketCounters, cnt.Rtt)
}

// Counters contains consumer counters.
type Counters struct {
	PacketCounters
	NAllocError uint64            `json:"nAllocError"`
	Rtt         RttCounters       `json:"rtt"`
	PerPattern  []PatternCounters `json:"perPattern"`
}

func (cnt Counters) String() string {
	s := fmt.Sprintf("%s %dalloc-error rtt=%s", cnt.PacketCounters, cnt.NAllocError, cnt.Rtt)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

// ReadCounters retrieves counters.
func (consumer *Consumer) ReadCounters() (cnt Counters) {
	rttScale := eal.GetNanosInTscUnit()
	for i := 0; i < int(consumer.rxC.nPatterns); i++ {
		crP := consumer.rxC.pattern[i]
		ctP := consumer.txC.pattern[i]
		rtt := runningstat.FromPtr(unsafe.Pointer(&crP.rtt)).Read().Scale(rttScale)

		var pcnt PatternCounters
		pcnt.NInterests = uint64(ctP.nInterests)
		pcnt.NData = uint64(crP.nData)
		pcnt.NNacks = uint64(crP.nNacks)
		pcnt.Rtt.Snapshot = rtt
		cnt.PerPattern = append(cnt.PerPattern, pcnt)

		cnt.NInterests += pcnt.NInterests
		cnt.NData += pcnt.NData
		cnt.NNacks += pcnt.NNacks
		cnt.Rtt.Snapshot = cnt.Rtt.Snapshot.Add(rtt)
	}

	cnt.NAllocError = uint64(consumer.txC.nAllocError)
	return cnt
}

// ClearCounters clears counters.
// Both RX and TX threads should be stopped before calling this, otherwise race conditions may occur.
func (consumer *Consumer) ClearCounters() {
	nPatterns := int(consumer.rxC.nPatterns)
	for i := 0; i < nPatterns; i++ {
		consumer.clearCounter(i)
	}
}

func (consumer *Consumer) clearCounter(index int) {
	consumer.rxC.pattern[index].nData = 0
	consumer.rxC.pattern[index].nNacks = 0
	rtt := runningstat.FromPtr(unsafe.Pointer(&consumer.rxC.pattern[index].rtt))
	rtt.Clear(true)
	consumer.txC.pattern[index].nInterests = 0
}
