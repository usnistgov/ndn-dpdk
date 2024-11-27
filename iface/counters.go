package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"
	"reflect"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

// RxCounters contains face/queue RX counters.
type RxCounters struct {
	RxFrames    uint64 `json:"rxFrames" gqldesc:"RX total frames."`
	RxOctets    uint64 `json:"rxOctets" gqldesc:"RX total bytes."`
	RxInterests uint64 `json:"rxInterests" gqldesc:"RX Interest packets."`
	RxData      uint64 `json:"rxData" gqldesc:"RX Data packets."`
	RxNacks     uint64 `json:"rxNacks" gqldesc:"RX Nack packets."`

	RxDecodeErrs   uint64 `json:"rxDecodeErrs" gqldesc:"RX decode errors."`
	RxReassPackets uint64 `json:"rxReassPackets" gqldesc:"RX packets that were reassembled."`
	RxReassDrops   uint64 `json:"rxReassDrops" gqldesc:"RX frames that were dropped by reassembler."`
}

func (cnt RxCounters) String() string {
	return fmt.Sprintf("%dfrm %db %dI %dD %dN %derr reass=(%dpkt %ddrop)",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.RxDecodeErrs, cnt.RxReassPackets, cnt.RxReassDrops)
}

func (cnt *RxCounters) readFrom(c *C.FaceRxThread) {
	cnt.RxOctets = uint64(c.nFrames[C.FaceRxThread_cntNOctets])
	cnt.RxInterests = uint64(c.nFrames[ndni.PktInterest])
	cnt.RxData = uint64(c.nFrames[ndni.PktData])
	cnt.RxNacks = uint64(c.nFrames[ndni.PktNack])

	cnt.RxDecodeErrs = uint64(c.nDecodeErr)
	cnt.RxReassPackets = uint64(c.reass.nDeliverPackets)
	cnt.RxReassDrops = uint64(c.reass.nDropFragments)

	cnt.RxFrames = cnt.RxInterests + cnt.RxData + cnt.RxNacks - cnt.RxReassPackets + uint64(c.reass.nDeliverFragments) + cnt.RxReassDrops
}

// TxCounters contains face/queue TX counters.
type TxCounters struct {
	TxFrames    uint64 `json:"txFrames" gqldesc:"TX total frames."`
	TxOctets    uint64 `json:"txOctets" gqldesc:"TX total bytes."`
	TxInterests uint64 `json:"txInterests" gqldesc:"TX Interest packets."`
	TxData      uint64 `json:"txData" gqldesc:"TX Data packets."`
	TxNacks     uint64 `json:"txNacks" gqldesc:"TX Nack packets."`

	TxFragGood  uint64 `json:"txFragGood" gqldesc:"TX fragmented L3 packets."`
	TxFragBad   uint64 `json:"txFragBad" gqldesc:"TX fragmentation failures."`
	TxAllocErrs uint64 `json:"txAllocErrs" gqldesc:"TX allocation errors."`
	TxDropped   uint64 `json:"txDropped" gqldesc:"TX dropped L2 frames due to full queue."`
}

func (cnt TxCounters) String() string {
	return fmt.Sprintf("%dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %ddropped",
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.TxFragGood, cnt.TxFragBad, cnt.TxAllocErrs, cnt.TxDropped)
}

func (cnt *TxCounters) readFrom(c *C.FaceTxThread) {
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
			zeroIndex = min(zeroIndex, i)
		} else {
			for field := range sumV.NumField() {
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
		return
	}

	for _, rxt := range c.impl.rx {
		var rxCnt RxCounters
		rxCnt.readFrom(&rxt)
		cnt.RxThreads = append(cnt.RxThreads, rxCnt)
	}
	cnt.sumRx()

	cnt.TxCounters.readFrom(&c.impl.tx[0])

	return cnt
}
