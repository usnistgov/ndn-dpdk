package ndni_test

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndntestenv"
)

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndntestenv.MakeInterest
	makeData     = ndntestenv.MakeData
)

func packetFromHex(input string) *ndni.Packet {
	return ndni.PacketFromPtr(mbuftestenv.MakePacket(input).GetPtr())
}

func tlvBytesFromHex(input string) ndni.TlvBytes {
	return ndni.TlvBytes(mbuftestenv.BytesFromHex(input))
}
