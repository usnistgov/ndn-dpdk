package ifacetest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	directMp := dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	indirectMp := dpdktestenv.MakeIndirectMp(255)

	mockface.FaceMempools.IndirectMp = indirectMp
	mockface.FaceMempools.HeaderMp = directMp
	mockface.FaceMempools.NameMp = directMp

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
