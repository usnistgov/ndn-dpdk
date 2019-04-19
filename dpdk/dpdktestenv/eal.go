package dpdktestenv

import (
	"os"

	"ndn-dpdk/dpdk"
)

func InitEal() {
	args := []string{"testprog", "-c0x3f", "-n1"}
	if os.Getenv("DPDKTESTENV_PCI") != "1" {
		args = append(args, "--no-pci")
	}
	dpdk.MustInitEal(args)
}
