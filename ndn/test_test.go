package ndn_test

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	mp := dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	parseMps.MpName = mp

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

func packetFromHex(input string) ndn.Packet {
	return ndn.PacketFromPtr(dpdktestenv.PacketFromHex(input).GetPtr())
}

func TlvBytesFromHex(input string) ndn.TlvBytes {
	return ndn.TlvBytes(dpdktestenv.PacketBytesFromHex(input))
}

var parseMps ndn.PacketParseMempools
