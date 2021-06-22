package ealconfig

import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
)

// ErrNoLCore indicates there is no LCore available.
var ErrNoLCore = errors.New("no LCore available")

// LCoreConfig contains CPU and logical core related configuration.
type LCoreConfig struct {
	// Cores is the list of processors (hardware cores) available to DPDK.
	// Note that Go code is not restricted to these cores.
	//
	// The default is allowing all cores on the system.
	// If this list contains a non-existent core, it is skipped.
	Cores []int `json:"cores,omitempty"`

	// CoresPerNuma maps from NUMA socket ID to the number of cores available to DPDK.
	// This is ignored if Cores is specified.
	//
	// Example:
	//  CoresPerNuma[0] = 10     allows up to 10 cores on socket 0.
	//  CoresPerNuma[1] = -2     allows all but 2 cores on socket 1.
	//  CoresPerNuma[2] = 0      disallows all cores on socket 2.
	//  Omitting CoresPerNuma[3] allows all cores on socket 3.
	//
	// If this map contains a non-existent NUMA socket, it is skipped.
	CoresPerNuma map[int]int `json:"coresPerNuma,omitempty"`

	// LCoreFlags is lcore-related flags passed to DPDK.
	// This replaces all other options.
	LCoreFlags string `json:"lcoreFlags,omitempty"`
}

func (cfg LCoreConfig) args(req Request, hwInfo hwinfo.Provider) (args []string, e error) {
	if cfg.LCoreFlags != "" {
		return shellSplit("LCoreFlags", cfg.LCoreFlags)
	}

	cores := hwInfo.Cores()
	var coreList commaSeparatedNumbers
	if len(cfg.Cores) > 0 {
		for _, coreID := range cfg.Cores {
			if cores.HasLogicalCore(coreID) {
				coreList.Append(coreID)
			}
		}
	} else {
		for socket, maxSocket := 0, cores.MaxNumaSocket(); socket <= maxSocket; socket++ {
			cfg.pickFromSocket(cores, socket, &coreList)
		}
	}

	if nCores := len(coreList); nCores == 0 {
		return nil, ErrNoLCore
	} else if nCores >= req.MinLCores {
		return []string{"-l", coreList.String()}, nil
	}
	return []string{"--lcores", fmt.Sprintf("(0-%d)@(%s)", req.MinLCores-1, coreList)}, nil
}

func (cfg LCoreConfig) pickFromSocket(cores hwinfo.Cores, socket int, coreList *commaSeparatedNumbers) {
	socketCores := append([]int{}, cores.ListPrimary(socket)...)
	socketCores = append(socketCores, cores.ListSecondary(socket)...)
	pref, hasPref := cfg.CoresPerNuma[socket]

	switch {
	case !hasPref: // allow all cores
	case pref == 0: // disallow all cores
		socketCores = nil
	case pref < 0: // disallow some cores
		pref += len(socketCores)
		if pref <= 0 { // all cores disallowed
			socketCores = nil
			break
		}
		fallthrough
	case pref > 0: // allow some cores
		if len(socketCores) > pref {
			socketCores = socketCores[:pref]
		}
	}

	for _, coreID := range socketCores {
		coreList.Append(coreID)
	}
}
