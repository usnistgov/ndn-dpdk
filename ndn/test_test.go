package ndn

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(255, 0, 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

func packetFromHex(input string) dpdk.Packet {
	return dpdktestenv.PacketFromHex(input)
}
