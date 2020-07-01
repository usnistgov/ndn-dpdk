package pktmbuf_test

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	makePacket   = mbuftestenv.MakePacket
)
