package mbuftestenv

/*
#include "../../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"encoding/hex"
	"fmt"
	"strings"

	"ndn-dpdk/dpdk/pktmbuf"
)

// BytesFromHex converts a hexadecimal string to a byte slice.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered comments and stripped.
func BytesFromHex(input string) []byte {
	s := strings.Map(func(ch rune) rune {
		if strings.ContainsRune("0123456789ABCDEF", ch) {
			return ch
		}
		return -1
	}, input)
	decoded, e := hex.DecodeString(s)
	if e != nil {
		panic(fmt.Sprintf("hex.DecodeString error %v", e))
	}
	return decoded
}

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
			segments = append(segments, BytesFromHex(a))
		case []string:
			for _, hexString := range a {
				segments = append(segments, BytesFromHex(hexString))
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
		mp = Direct.Pool()
	}
	if len(segments) == 0 {
		return mp.MustAlloc(1)[0]
	}

	vec := mp.MustAlloc(len(segments))
	pkt = vec[0]
	for i, b := range segments {
		seg := vec[i]
		if headroom != nil {
			seg.SetHeadroom(int(*headroom))
		}
		seg.Append(b)
		if i > 0 {
			pkt.Chain(seg)
		}
	}
	return pkt
}
