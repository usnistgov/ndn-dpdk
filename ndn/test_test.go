package ndn

// This file contains test setup procedure and common test helper functions.

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
)

var testEal *dpdk.Eal
var testMp dpdk.PktmbufPool

const testMp_DATAROOM = 2000

var testMpIndirect dpdk.PktmbufPool

func TestMain(m *testing.M) {
	eal, e := dpdk.NewEal([]string{"testprog", "-n1"})
	if e != nil || eal == nil {
		panic(fmt.Sprintf("NewEal error %v", e))
	}

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
	buf, e := seg0.Append(len(input))
	if e != nil {
		panic(e)
	}

	for i, b := range input {
		ptr := unsafe.Pointer(uintptr(buf) + uintptr(i))
		*(*byte)(ptr) = b
	}

	return pkt
}

func packetFromHex(input string) dpdk.Packet {
	decoded, e := hex.DecodeString(strings.Replace(input, " ", "", -1))
	if e != nil {
		return dpdk.Packet{}
	}
	return packetFromBytes(decoded)
}