package ndn_test

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndntestenv.MakeInterest
	makeData     = ndntestenv.MakeData
)

func packetFromHex(input string) *ndn.Packet {
	return ndn.PacketFromPtr(mbuftestenv.MakePacket(input).GetPtr())
}

func tlvBytesFromHex(input string) ndn.TlvBytes {
	return ndn.TlvBytes(mbuftestenv.BytesFromHex(input))
}
