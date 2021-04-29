package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Counters contains basic face counters.
type Counters struct {
	RxFrames uint64 `json:"rxFrames"` // RX total frames
	RxOctets uint64 `json:"rxOctets"` // RX total bytes

	DecodeErrs   uint64 `json:"decodeErrs"`   // decode errors
	ReassPackets uint64 `json:"reassPackets"` // RX packets that were reassembled
	ReassDrops   uint64 `json:"reassDrops"`   // RX frames that were dropped by reassembler

	RxInterests uint64 `json:"rxInterests"` // RX Interest packets
	RxData      uint64 `json:"rxData"`      // RX Data packets
	RxNacks     uint64 `json:"rxNacks"`     // RX Nack packets

	TxInterests uint64 `json:"txInterests"` // TX Interest packets
	TxData      uint64 `json:"txData"`      // TX Data packets
	TxNacks     uint64 `json:"txNacks"`     // TX Nack packets

	FragGood    uint64 `json:"fragGood"`    // fragmented L3 packets
	FragBad     uint64 `json:"fragBad"`     // fragmentation failures
	TxAllocErrs uint64 `json:"txAllocErrs"` // allocation errors during TX
	TxDropped   uint64 `json:"txDropped"`   // L2 frames dropped due to full queue
	TxFrames    uint64 `json:"txFrames"`    // sent total frames
	TxOctets    uint64 `json:"txOctets"`    // sent total bytes
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %dfrm %db %dI %dD %dN %derr reass=(%dpkt %ddrop) TX %dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %ddropped",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.DecodeErrs, cnt.ReassPackets, cnt.ReassDrops,
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.FragGood, cnt.FragBad, cnt.TxAllocErrs, cnt.TxDropped)
}

// Counters retrieves basic face counters.
func (f *face) Counters() (cnt Counters) {
	c := f.ptr()
	if c.impl == nil {
		return cnt
	}

	rxC := &c.impl.rx
	for i := 0; i < C.RXPROC_MAX_THREADS; i++ {
		rxtC := &rxC.threads[i]
		cnt.RxOctets += uint64(rxtC.nFrames[0])
		cnt.DecodeErrs += uint64(rxtC.nDecodeErr)
		cnt.RxInterests += uint64(rxtC.nFrames[ndni.PktInterest])
		cnt.RxData += uint64(rxtC.nFrames[ndni.PktData])
		cnt.RxNacks += uint64(rxtC.nFrames[ndni.PktNack])
	}
	cnt.ReassPackets = uint64(rxC.reass.nDeliverPackets)
	cnt.ReassDrops = uint64(rxC.reass.nDropFragments)
	cnt.RxFrames = cnt.RxInterests + cnt.RxData + cnt.RxNacks + uint64(rxC.reass.nDeliverFragments) - cnt.ReassPackets + cnt.ReassDrops

	txC := &c.impl.tx
	cnt.TxInterests = uint64(txC.nFrames[ndni.PktInterest])
	cnt.TxData = uint64(txC.nFrames[ndni.PktData])
	cnt.TxNacks = uint64(txC.nFrames[ndni.PktNack])

	cnt.FragGood = uint64(txC.nL3Fragmented)
	cnt.FragBad = uint64(txC.nL3OverLength + txC.nAllocFails)
	cnt.TxAllocErrs = uint64(txC.nAllocFails)
	cnt.TxDropped = uint64(txC.nDroppedFrames)
	cnt.TxFrames = uint64(txC.nFrames[ndni.PktFragment] - txC.nDroppedFrames)
	cnt.TxOctets = uint64(txC.nOctets - txC.nDroppedOctets)

	return cnt
}
