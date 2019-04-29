package ifacetestfixture

import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

func MakeMempools() (rxMp dpdk.PktmbufPool, mempools iface.Mempools) {
	rxMp = dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 5000)
	mempools = iface.Mempools{
		IndirectMp: dpdktestenv.MakeIndirectMp(4095),
		NameMp:     dpdktestenv.MakeMp("name", 4095, 0, ndn.NAME_MAX_LENGTH),
		HeaderMp:   dpdktestenv.MakeMp("header", 4095, 0, ethface.SizeofTxHeader()),
	}
	return
}
