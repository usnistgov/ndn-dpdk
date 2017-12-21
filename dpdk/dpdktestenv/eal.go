package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var Eal *dpdk.Eal

func InitEal() *dpdk.Eal {
	if Eal != nil {
		return Eal
	}

	var e error
	Eal, e = dpdk.NewEal([]string{"testprog", "-n1", "--no-pci"})
	if e != nil || Eal == nil {
		panic(fmt.Sprintf("dpdk.NewEal error %v", e))
	}
	return Eal
}
