package dpdktestenv

import (
	"ndn-dpdk/dpdk"
)

func InitEal() {
	dpdk.MustInitEal([]string{"testprog", "-n1", "--no-pci"})
}
