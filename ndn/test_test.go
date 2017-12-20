package ndn

// This file contains test setup procedure and common test helper functions.

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

var testMp dpdk.PktmbufPool

const testMp_DATAROOM = 2000

var testMpIndirect dpdk.PktmbufPool

func TestMain(m *testing.M) {
	dpdktestenv.InitEal()

	var e error
	testMp, e = dpdk.NewPktmbufPool("MP", 255, 0, 0, testMp_DATAROOM, dpdk.NUMA_SOCKET_ANY)
	if e != nil {
		panic(fmt.Sprintf("NewPktmbufPool(MP) error %v", e))
	}

	testMpIndirect, e = dpdk.NewPktmbufPool("MP-INDIRECT", 255, 0, 0, 0, dpdk.NUMA_SOCKET_ANY)
	if e != nil {
		panic(fmt.Sprintf("NewPktmbufPool(MP-INDIRECT) error %v", e))
	}

	os.Exit(m.Run())
}

func makeAR(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

// Make packet from raw bytes.
// Memory is allocated from testMp.
// Caller is responsible for closing the packet.
func packetFromBytes(input []byte) dpdk.Packet {
	m, e := testMp.Alloc()
	if e != nil {
		return dpdk.Packet{}
	}

	pkt := m.AsPacket()
	seg0 := pkt.GetFirstSegment()
	e = seg0.AppendOctets(input)
	if e != nil {
		panic(e)
	}

	return pkt
}

// Make packet from hexadecimal string.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered as comments and stripped.
func packetFromHex(input string) dpdk.Packet {
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
	return packetFromBytes(decoded)
}
