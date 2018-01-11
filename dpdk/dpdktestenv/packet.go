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
	m := Alloc(MPID_DIRECT)

	pkt := m.AsPacket()
	seg0 := pkt.GetFirstSegment()
	e := seg0.AppendOctets(input)
	if e != nil {
		panic(fmt.Sprintf("Segment.AppendOctets error %v, packet too long?", e))
	}

	return pkt
}

func PacketBytesFromHex(input string) []byte {
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

// Make packet from hexadecimal string.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered as comments and stripped.
func PacketFromHex(input string) dpdk.Packet {
	bytes := PacketBytesFromHex(input)
	if bytes == nil {
		return dpdk.Packet{}
	}
	return PacketFromBytes(bytes)
}
