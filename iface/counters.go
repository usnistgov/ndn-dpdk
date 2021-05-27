package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"
	"reflect"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// RxCounters contains face/queue RX counters.
type RxCounters struct {
	RxFrames    uint64 `json:"rxFrames"`    // RX total frames
	RxOctets    uint64 `json:"rxOctets"`    // RX total bytes
	RxInterests uint64 `json:"rxInterests"` // RX Interest packets
	RxData      uint64 `json:"rxData"`      // RX Data packets
	RxNacks     uint64 `json:"rxNacks"`     // RX Nack packets

	RxDecodeErrs   uint64 `json:"rxDecodeErrs"`   // decode errors
	RxReassPackets uint64 `json:"rxReassPackets"` // RX packets that were reassembled
	RxReassDrops   uint64 `json:"rxReassDrops"`   // RX frames that were dropped by reassembler
}

func (cnt RxCounters) String() string {
	return fmt.Sprintf("%dfrm %db %dI %dD %dN %derr reass=(%dpkt %ddrop)",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.RxDecodeErrs, cnt.RxReassPackets, cnt.RxReassDrops)
}

func (cnt *RxCounters) readFrom(c *C.RxProcThread) {
	cnt.RxOctets = uint64(c.nFrames[0])
	cnt.RxInterests = uint64(c.nFrames[ndni.PktInterest])
	cnt.RxData = uint64(c.nFrames[ndni.PktData])
	cnt.RxNacks = uint64(c.nFrames[ndni.PktNack])

	cnt.RxDecodeErrs = uint64(c.nDecodeErr)
	cnt.RxReassPackets = uint64(c.reass.nDeliverPackets)
	cnt.RxReassDrops = uint64(c.reass.nDropFragments)

	cnt.RxFrames = cnt.RxInterests + cnt.RxData + cnt.RxNacks - cnt.RxReassPackets + uint64(c.reass.nDeliverFragments) + cnt.RxReassDrops
}

// RxCounters contains face/queue TX counters.
type TxCounters struct {
	TxFrames    uint64 `json:"txFrames"`    // sent total frames
	TxOctets    uint64 `json:"txOctets"`    // sent total bytes
	TxInterests uint64 `json:"txInterests"` // TX Interest packets
	TxData      uint64 `json:"txData"`      // TX Data packets
	TxNacks     uint64 `json:"txNacks"`     // TX Nack packets

	TxFragGood  uint64 `json:"txFragGood"`  // fragmented L3 packets
	TxFragBad   uint64 `json:"txFragBad"`   // fragmentation failures
	TxAllocErrs uint64 `json:"txAllocErrs"` // allocation errors during TX
	TxDropped   uint64 `json:"txDropped"`   // L2 frames dropped due to full queue
}

func (cnt TxCounters) String() string {
	return fmt.Sprintf("%dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %ddropped",
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.TxFragGood, cnt.TxFragBad, cnt.TxAllocErrs, cnt.TxDropped)
}

func (cnt *TxCounters) readFrom(c *C.TxProc) {
	cnt.TxFrames = uint64(c.nFrames[ndni.PktFragment] - c.nDroppedFrames)
	cnt.TxOctets = uint64(c.nOctets - c.nDroppedOctets)
	cnt.TxInterests = uint64(c.nFrames[ndni.PktInterest])
	cnt.TxData = uint64(c.nFrames[ndni.PktData])
	cnt.TxNacks = uint64(c.nFrames[ndni.PktNack])

	cnt.TxFragGood = uint64(c.nL3Fragmented)
	cnt.TxFragBad = uint64(c.nL3OverLength + c.nAllocFails)
	cnt.TxAllocErrs = uint64(c.nAllocFails)
	cnt.TxDropped = uint64(c.nDroppedFrames)
}

// Counters contains face counters.
type Counters struct {
	RxCounters
	TxCounters

	RxThreads []RxCounters `json:"rxThreads"`
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %s TX %s", cnt.RxCounters, cnt.TxCounters)
}

func (cnt *Counters) sumRx() {
	zeroIndex := len(cnt.RxThreads)
	sumV := reflect.ValueOf(&cnt.RxCounters).Elem()
	for i, thCnt := range cnt.RxThreads {
		thV := reflect.ValueOf(thCnt)
		if thV.IsZero() {
			zeroIndex = math.MinInt(zeroIndex, i)
		} else {
			for field, nFields := 0, sumV.NumField(); field < nFields; field++ {
				sumF, thF := sumV.Field(field), thV.Field(field)
				sumF.SetUint(sumF.Uint() + thF.Uint())
			}
		}
	}
	cnt.RxThreads = cnt.RxThreads[:zeroIndex]
}

// Counters retrieves face counters.
func (f *face) Counters() (cnt Counters) {
	c := f.ptr()
	if c.impl == nil {
		return cnt
	}

	rxC := &c.impl.rx
	for _, rxtC := range rxC.threads {
		var rxCnt RxCounters
		rxCnt.readFrom(&rxtC)
		cnt.RxThreads = append(cnt.RxThreads, rxCnt)
	}
	cnt.sumRx()

	txC := &c.impl.tx
	cnt.TxCounters.readFrom(txC)

	return cnt
}
