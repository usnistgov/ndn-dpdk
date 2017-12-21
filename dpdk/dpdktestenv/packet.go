package dpdktestenv

import (
	"encoding/hex"
	"fmt"
	"strings"

	"ndn-dpdk/dpdk"
)

// Make packet from raw bytes.
// Memory is allocated from DirectMp.
// Caller is responsible for closing the packet.
func PacketFromBytes(input []byte) dpdk.Packet {
	m, e := DirectMp.Alloc()
	if e != nil {
		panic(fmt.Sprintf("PacketFromBytes error %v", e))
	}

	pkt := m.AsPacket()
	seg0 := pkt.GetFirstSegment()
	e = seg0.AppendOctets(input)
	if e != nil {
		panic(fmt.Sprintf("Segment.AppendOctets error %v, packet too long?", e))
	}

	return pkt
}

// Make packet from hexadecimal string.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered as comments and stripped.
func PacketFromHex(input string) dpdk.Packet {
	s := strings.Map(func(ch rune) rune {
		if strings.ContainsRune("0123456789ABCDEF", ch) {
			return ch
		}
		return -1
	}, input)
	decoded, e := hex.DecodeString(s)
	if e != nil {
		return dpdk.Packet{}
	}
	return PacketFromBytes(decoded)
}
