package ealtestenv

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
)

// EnvCpus declares an environment variable to reduce the number of CPU cores to use.
// This allows running tests on fewer CPU cores.
const EnvCpus = "EALTESTENV_CPUS"

// EnvPci declares an environment variable that, when set to 1, enables the use of PCI devices.
// The default is disabling PCI devices.
const EnvPci = "EALTESTENV_PCI"

// WantLCores indicates the number of lcores to be created.
var WantLCores = 6

// UsingThreads is set to true if there are fewer CPU cores than lcores.
var UsingThreads = false

// Init initializes EAL for testing purpose.
func Init(extraArgs ...string) (remainingArgs []string) {
	args := []string{"testprog", "-n1"}
	args = append(args, pickCpus()...)

	if os.Getenv(EnvPci) != "1" {
		args = append(args, "--no-pci")
	}

	args = append(args, extraArgs...)
	return ealinit.Init(args)
}

func listCpus() (primary, secondary []int) {
	cpu, e := ghw.CPU()
	if e != nil {
		panic(e)
	}
	for _, processor := range cpu.Processors {
		for _, core := range processor.Cores {
			for i, thread := range core.LogicalProcessors {
				if i == 0 {
					primary = append(primary, thread)
				} else {
					secondary = append(secondary, thread)
				}
			}
		}
	}
	return
}

func pickCpus() (lcoresArg []string) {
	primary, secondary := listCpus()
	shuffleInts(primary)
	shuffleInts(secondary)
	allCpus := append(append([]int{}, primary...), secondary...)

	useCpus := allCpus
	if limit, _ := strconv.Atoi(os.Getenv(EnvCpus)); limit > 0 && limit < len(allCpus) {
		useCpus = allCpus[:limit]
	}

	if len(useCpus) < WantLCores {
		UsingThreads = true
		return []string{"--lcores", fmt.Sprintf("(0-%d)@(%s)", WantLCores-1, sprintInts(useCpus))}
	}
	return []string{"-l" + sprintInts(useCpus[:WantLCores])}
}

func shuffleInts(a []int) {
	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
}

func sprintInts(a []int) string {
	var w strings.Builder
	delim := ""
	for _, i := range a {
		w.WriteString(delim)
		delim = ","
		w.Write(strconv.AppendInt(nil, int64(i), 10))
	}
	return w.String()
}
