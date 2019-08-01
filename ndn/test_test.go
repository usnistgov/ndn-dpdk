package ndn_test

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

var theMp dpdk.PktmbufPool

func TestMain(m *testing.M) {
	theMp = dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	dpdktestenv.MakeIndirectMp(255)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

func packetFromHex(input string) ndn.Packet {
	return ndn.PacketFromPtr(dpdktestenv.PacketFromHex(input).GetPtr())
}

func TlvBytesFromHex(input string) ndn.TlvBytes {
	return ndn.TlvBytes(dpdktestenv.BytesFromHex(input))
}
