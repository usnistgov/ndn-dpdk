package pingclient

/*
#include "rx.h"
#include "token.h"
#include "tx.h"
*/
import "C"
import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
)

type PacketCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64
}

func (cnt PacketCounters) ComputeDataRatio() float64 {
	return float64(cnt.NData) / float64(cnt.NInterests)
}

func (cnt PacketCounters) ComputeNackRatio() float64 {
	return float64(cnt.NNacks) / float64(cnt.NInterests)
}

func (cnt PacketCounters) String() string {
	return fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%)",
		cnt.NInterests,
		cnt.NData, cnt.ComputeDataRatio()*100.0,
		cnt.NNacks, cnt.ComputeNackRatio()*100.0)
}

type RttCounters struct {
	running_stat.Snapshot
}

func (cnt RttCounters) String() string {
	ms := cnt.Scale(1.0 / float64(time.Millisecond))
	return fmt.Sprintf("%0.3f/%0.3f/%0.3f/%0.3fms", ms.Min(), ms.Mean(), ms.Max(), ms.Stdev())
}

type PatternCounters struct {
	PacketCounters
	Rtt         RttCounters
	NRttSamples uint64
}

func (cnt PatternCounters) String() string {
	return fmt.Sprintf("%s rtt=%s(%dsamp)",
		cnt.PacketCounters, cnt.Rtt, cnt.NRttSamples)
}

type Counters struct {
	PacketCounters
	NAllocError uint64
	Rtt         RttCounters
	PerPattern  []PatternCounters
}

func (cnt Counters) String() string {
	s := fmt.Sprintf("%s %dalloc-error rtt=%s", cnt.PacketCounters, cnt.NAllocError, cnt.Rtt)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

// Read counters.
func (client *Client) ReadCounters() (cnt Counters) {
	rttScale := dpdk.GetNanosInTscUnit() * math.Exp2(C.PING_TIMING_PRECISION)
	var rttCombined running_stat.Snapshot
	for i := 0; i < int(client.Rx.c.nPatterns); i++ {
		crP := client.Rx.c.pattern[i]
		ctP := client.Tx.c.pattern[i]
		rtt := running_stat.FromPtr(unsafe.Pointer(&crP.rtt)).Read().Scale(rttScale)

		var pcnt PatternCounters
		pcnt.NInterests = uint64(ctP.nInterests)
		pcnt.NData = uint64(crP.nData)
		pcnt.NNacks = uint64(crP.nNacks)
		pcnt.Rtt.Snapshot = rtt
		cnt.PerPattern = append(cnt.PerPattern, pcnt)

		cnt.NInterests += pcnt.NInterests
		cnt.NData += pcnt.NData
		cnt.NNacks += pcnt.NNacks
		rttCombined.Combine(rtt)
	}

	cnt.NAllocError = uint64(client.Tx.c.nAllocError)
	cnt.Rtt.Snapshot = rttCombined
	return cnt
}

// Clear counters. Both RX and TX threads should be stopped before calling this,
// otherwise race conditions may occur.
func (client *Client) ClearCounters() {
	nPatterns := int(client.Rx.c.nPatterns)
	for i := 0; i < nPatterns; i++ {
		client.clearCounter(i)
	}
}

func (client *Client) clearCounter(index int) {
	client.Rx.c.pattern[index].nData = 0
	client.Rx.c.pattern[index].nNacks = 0
	rtt := running_stat.FromPtr(unsafe.Pointer(&client.Rx.c.pattern[index].rtt))
	rtt.Clear(true)
	client.Tx.c.pattern[index].nInterests = 0
}
