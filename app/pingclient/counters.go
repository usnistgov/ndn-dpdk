package pingclient

/*
#include "../../csrc/pingclient/rx.h"
#include "../../csrc/pingclient/token.h"
#include "../../csrc/pingclient/tx.h"
*/
import "C"
import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
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
	runningstat.Snapshot
}

func (cnt RttCounters) String() string {
	ms := cnt.Scale(1.0 / float64(time.Millisecond))
	return fmt.Sprintf("%0.3f/%0.3f/%0.3f/%0.3fms(%dsamp)", ms.Min(), ms.Mean(), ms.Max(), ms.Stdev(), ms.Len())
}

type PatternCounters struct {
	PacketCounters
	Rtt         RttCounters
	NRttSamples uint64
}

func (cnt PatternCounters) String() string {
	return fmt.Sprintf("%s rtt=%s", cnt.PacketCounters, cnt.Rtt)
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
	rttScale := eal.GetNanosInTscUnit() * math.Exp2(C.PING_TIMING_PRECISION)
	for i := 0; i < int(client.rxC.nPatterns); i++ {
		crP := client.rxC.pattern[i]
		ctP := client.txC.pattern[i]
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

	cnt.NAllocError = uint64(client.txC.nAllocError)
	return cnt
}

// Clear counters. Both RX and TX threads should be stopped before calling this,
// otherwise race conditions may occur.
func (client *Client) ClearCounters() {
	nPatterns := int(client.rxC.nPatterns)
	for i := 0; i < nPatterns; i++ {
		client.clearCounter(i)
	}
}

func (client *Client) clearCounter(index int) {
	client.rxC.pattern[index].nData = 0
	client.rxC.pattern[index].nNacks = 0
	rtt := runningstat.FromPtr(unsafe.Pointer(&client.rxC.pattern[index].rtt))
	rtt.Clear(true)
	client.txC.pattern[index].nInterests = 0
}
