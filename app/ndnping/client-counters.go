package ndnping

/*
#include "client.h"
#include "token.h"
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

type ClientPatternCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NRttSamples uint64
	RttMin      time.Duration
	RttMax      time.Duration
	RttAvg      time.Duration
	RttStdev    time.Duration
}

func (cnt ClientPatternCounters) String() string {
	return fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%) rtt=%0.3f/%0.3f/%0.3f/%0.3fms(%dsamp)",
		cnt.NInterests,
		cnt.NData, float64(cnt.NData)/float64(cnt.NInterests)*100.0,
		cnt.NNacks, float64(cnt.NNacks)/float64(cnt.NInterests)*100.0,
		float64(cnt.RttMin)/float64(time.Millisecond), float64(cnt.RttAvg)/float64(time.Millisecond),
		float64(cnt.RttMax)/float64(time.Millisecond), float64(cnt.RttStdev)/float64(time.Millisecond),
		cnt.NRttSamples)
}

type ClientCounters struct {
	PerPattern  []ClientPatternCounters
	NInterests  uint64
	NData       uint64
	NNacks      uint64
	NAllocError uint64
}

func (cnt ClientCounters) ComputeRatios() (dataRatio, nackRatio float64) {
	dataRatio = float64(cnt.NData) / float64(cnt.NInterests)
	nackRatio = float64(cnt.NNacks) / float64(cnt.NInterests)
	return
}

func (cnt ClientCounters) String() string {
	dataRatio, nackRatio := cnt.ComputeRatios()
	s := fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%) %dalloc-error", cnt.NInterests,
		cnt.NData, dataRatio*100.0, cnt.NNacks, nackRatio*100.0, cnt.NAllocError)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

// Read counters.
func (client *Client) ReadCounters() (cnt ClientCounters) {
	durationUnit := dpdk.GetNanosInTscUnit() *
		math.Pow(2.0, float64(C.NDNPING_TIMING_PRECISION))
	toDuration := func(d float64) time.Duration {
		return time.Duration(d * durationUnit)
	}

	patterns := client.getPatterns()
	cnt.PerPattern = make([]ClientPatternCounters, patterns.Len())
	for i := 0; i < len(cnt.PerPattern); i++ {
		pattern := (*C.NdnpingClientPattern)(patterns.GetUsr(i))
		rtt := running_stat.FromPtr(unsafe.Pointer(&pattern.rtt))
		perPattern := ClientPatternCounters{
			NInterests:  uint64(pattern.nInterests),
			NData:       uint64(pattern.nData),
			NNacks:      uint64(pattern.nNacks),
			NRttSamples: rtt.Len64(),
			RttMin:      toDuration(rtt.Min()),
			RttMax:      toDuration(rtt.Max()),
			RttAvg:      toDuration(rtt.Mean()),
			RttStdev:    toDuration(rtt.Stdev()),
		}
		cnt.PerPattern[i] = perPattern
		cnt.NInterests += perPattern.NInterests
		cnt.NData += perPattern.NData
		cnt.NNacks += perPattern.NNacks
	}

	cnt.NAllocError = uint64(client.c.nAllocError)
	return cnt
}

// Clear counters. Both RX and TX threads should be stopped before calling this,
// otherwise race conditions may occur.
func (client *Client) ClearCounters() {
	patterns := client.getPatterns()
	for i := 0; i < patterns.Len(); i++ {
		pattern := (*C.NdnpingClientPattern)(patterns.GetUsr(i))
		*pattern = C.NdnpingClientPattern{}
	}
}
