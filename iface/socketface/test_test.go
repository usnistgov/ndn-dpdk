package socketface_test

// This file contains test setup procedure and common test helper functions.

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

var directMp dpdk.PktmbufPool
var faceMempools iface.Mempools

func TestMain(m *testing.M) {
	directMp = dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	faceMempools.IndirectMp = dpdktestenv.MakeIndirectMp(4095)
	faceMempools.HeaderMp = dpdktestenv.MakeMp("header", 4095, 0,
		uint16(ndn.EncodeLpHeader_GetHeadroom()+ndn.EncodeLpHeader_GetTailroom()))
	faceMempools.NameMp = dpdktestenv.MakeMp("name", 4095, 0, uint16(ndn.NAME_MAX_LENGTH))

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
