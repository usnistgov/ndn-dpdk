package ndn_test

import (
	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestenv"
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
