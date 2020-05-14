package dpdktestenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"

	"ndn-dpdk/dpdk"
)

func InitEal(extraArgs ...string) (remainingArgs []string) {
	wantLcores := 6
	cpus, nCpus := listCpus(wantLcores)

	args := []string{"testprog", "-n1"}
	if nCpus < wantLcores {
		args = append(args, "--lcores", fmt.Sprintf("(0-%d)@(%s)", wantLcores-1, cpus))
	} else {
		args = append(args, "-l"+cpus)
	}

	if os.Getenv("DPDKTESTENV_PCI") != "1" {
		args = append(args, "--no-pci")
	}

	args = append(args, extraArgs...)
	return dpdk.MustInitEal(args)
}

func listCpus(max int) (cpus string, n int) {
	cpu, e := ghw.CPU()
	if e != nil {
		panic(e)
	}
	var threads []int
	for _, processor := range cpu.Processors {
		var secondThreads []int
		for _, core := range processor.Cores {
			for i, thread := range core.LogicalProcessors {
				if i == 0 {
					threads = append(threads, thread)
				} else {
					secondThreads = append(secondThreads, thread)
				}
			}
		}
		threads = append(threads, secondThreads...)
	}

	n = len(threads)
	if n > max {
		n = max
	}

	var list []string
	for i, thread := range threads {
		if i >= n {
			break
		}
		list = append(list, strconv.Itoa(thread))
	}
	return strings.Join(list, ","), n
}
