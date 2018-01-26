package socketface

// This file contains test setup procedure and common test helper functions.

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

var directMp, indirectMp, headerMp dpdk.PktmbufPool

func TestMain(m *testing.M) {
	directMp = dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	indirectMp = dpdktestenv.MakeIndirectMp(4095)
	headerMp = dpdktestenv.MakeMp("header", 4095, 0,
		uint16(ndn.EncodeLpHeaders_GetHeadroom()+ndn.EncodeLpHeaders_GetTailroom()))

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

func packetFromHex(input string) dpdk.Packet {
	return dpdktestenv.PacketFromHex(input)
}
