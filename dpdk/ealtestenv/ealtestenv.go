// Package ealtestenv initializes EAL for unit testing.
package ealtestenv

import (
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
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

type hwInfoLimitCores struct {
	ealconfig.HwInfoSource
	MaxCores int
	cores    []ealconfig.CoreInfo
}

func (hwInfo *hwInfoLimitCores) Cores() []ealconfig.CoreInfo {
	if len(hwInfo.cores) == 0 {
		cores := hwInfo.HwInfoSource.Cores()
		rand.Shuffle(len(cores), reflect.Swapper(cores))
		if len(cores) > hwInfo.MaxCores {
			cores = cores[:hwInfo.MaxCores]
		}
		hwInfo.cores = cores
	}
	return hwInfo.cores
}

// Init initializes EAL for unit testing.
func Init() {
	rand.Seed(time.Now().UnixNano())

	hwInfo := &hwInfoLimitCores{
		HwInfoSource: ealconfig.DefaultHwInfoSource(),
		MaxCores:     WantLCores,
	}
	if maxCores, e := strconv.Atoi(os.Getenv(EnvCpus)); e == nil {
		hwInfo.MaxCores = math.MinInt(hwInfo.MaxCores, maxCores)
	}

	var cfg ealconfig.Config
	cfg.AllPciDevices = os.Getenv(EnvPci) == "1"

	var req ealconfig.Request
	req.MinLCores = WantLCores

	args, e := cfg.Args(req, hwInfo)
	if e != nil {
		panic(e)
	}
	ealinit.Init(args)

	UsingThreads = len(hwInfo.Cores()) < 1+len(eal.Workers)
}
