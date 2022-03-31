// Package mbuftestenv contains helper functions to construct mbufs in test code.
package mbuftestenv

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Headroom sets segment headroom for MakePacket.
type Headroom int

// MakePacket creates a packet.
// *pktmbuf.Pool specifies where to allocate memory from; the default is the Direct pool.
// Headroom sets headroom in each segment.
// []byte or hexadecimal string becomes a segment.
// []string is flattened.
// Caller is responsible for releasing the packet.
func MakePacket(args ...interface{}) (pkt *pktmbuf.Packet) {
	var mp *pktmbuf.Pool
	var segments [][]byte
	var headroom *Headroom
	for i, arg := range args {
		switch a := arg.(type) {
		case []byte:
			segments = append(segments, a)
		case string:
			segments = append(segments, testenv.BytesFromHex(a))
		case []string:
			for _, hexString := range a {
				segments = append(segments, testenv.BytesFromHex(hexString))
			}
		case *pktmbuf.Pool:
			mp = a
		case Headroom:
			headroom = &a
		default:
			panic(fmt.Sprintf("MakePacket args[%d] invalid type %T", i, arg))
		}
	}

	if mp == nil {
		mp = DirectMempool()
	}
	if len(segments) == 0 {
		segments = append(segments, []byte{})
	}

	vec := mp.MustAlloc(len(segments))
	pkt = vec[0]
	for i, b := range segments {
		seg := vec[i]
		if headroom != nil {
			seg.SetHeadroom(int(*headroom))
		}
		if e := seg.Append(b); e != nil {
			panic(fmt.Errorf("seg.Append(%d): %w", len(b), e))
		}
		if i > 0 {
			pkt.Chain(seg)
		}
	}
	return pkt
}
