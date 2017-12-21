package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var Eal *dpdk.Eal

func InitEal() {
	if Eal != nil {
		return
	}

	var e error
	Eal, e = dpdk.NewEal([]string{"testprog", "-n1"})
	if e != nil || Eal == nil {
		panic(fmt.Sprintf("dpdk.NewEal error %v", e))
	}
}
