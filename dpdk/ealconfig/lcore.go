package ealconfig

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
)

// LCoreConfig contains CPU and logical core related configuration.
type LCoreConfig struct {
	// Cores is the list of processors (hardware cores) available to DPDK.
	// Note that Go code is not restricted to these cores.
	//
	// The default is allowing all cores, subject to CPU affinity configured in systemd or Docker.
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

	// LCoresPerNuma maps from NUMA socket ID to the number of lcores created in DPDK.
	//
	// This should be specified only if there aren't enough processors to activate and use NDN-DPDK.
	// For each NUMA socket, the specified number of lcores are created as threads, floating among
	// all available processors on that NUMA socket.
	// These lcores are numbered from 0 consecutively starting from the lowest numbered NUMA socket.
	// Note that using threads can lead to suboptimal performance.
	//
	// Example:
	//  - There are two NUMA sockets with these available processors: { 0: [2,3], 1: [5,6,7] }
	//  - LCoresPerNuma is specified as: { 0: 4, 1: 6 }
	//  - This would create these LCores on NUMA sockets: { 0: [0,1,2,3], 1: [4,5,6,7,8,9] }
	//
	// If there are already enough processors, this should be left empty.
	LCoresPerNuma map[int]int `json:"lcoresPerNuma,omitempty"`

	// LCoreMain is the DPDK main lcore ID.
	LCoreMain *int `json:"lcoreMain,omitempty"`

	// LCoreFlags is lcore-related flags passed to DPDK.
	// This replaces all other options.
	LCoreFlags string `json:"lcoreFlags,omitempty"`
}

func (cfg LCoreConfig) args(hwInfo hwinfo.Provider) (args []string, e error) {
	if cfg.LCoreFlags != "" {
		return shellSplit("lcoreFlags", cfg.LCoreFlags)
	}

	avail := cfg.gatherAvail(hwInfo)
	if len(cfg.LCoresPerNuma) == 0 {
		var l commaSeparated
		for _, cores := range avail {
			l.AppendInt(cores...)
		}
		if len(l) == 0 {
			return nil, errors.New("no processor available")
		}
		args = append(args, "-l", l.String())
	} else {
		demandSockets := []int{}
		for socket, demand := range cfg.LCoresPerNuma {
			if demand == 0 {
				return nil, fmt.Errorf("LCoresPerNuma[%d] should not be zero", socket)
			}
			if len(avail[socket]) == 0 {
				return nil, fmt.Errorf("no processor available on NUMA socket %d", socket)
			}
			demandSockets = append(demandSockets, socket)
		}
		sort.Ints(demandSockets)

		var lcores commaSeparated
		nextLCoreID := 0
		for _, socket := range demandSockets {
			var slcores commaSeparated
			for i := 0; i < cfg.LCoresPerNuma[socket]; i++ {
				slcores.AppendInt(nextLCoreID)
				nextLCoreID++
			}
			var savail commaSeparated
			savail.AppendInt(avail[socket]...)
			lcores = append(lcores, fmt.Sprintf("(%s)@(%s)", slcores, savail))
		}
		args = append(args, "--lcores", lcores.String())
	}

	if cfg.LCoreMain != nil {
		args = append(args, "--main-lcore", strconv.Itoa(*cfg.LCoreMain))
	}

	return args, nil
}

func (cfg LCoreConfig) gatherAvail(hwInfo hwinfo.Provider) (availBySocket map[int][]int) {
	availBySocket = map[int][]int{}
	if len(cfg.Cores) > 0 {
		hwCores := hwInfo.Cores().ByLogicalCore()
		for _, coreID := range cfg.Cores {
			if hwCore, found := hwCores[coreID]; found {
				availBySocket[hwCore.NumaSocket] = append(availBySocket[hwCore.NumaSocket], coreID)
			}
		}
	} else {
		for socket, hwCores := range hwInfo.Cores().ByNumaSocket() {
			socketCores := append(hwCores.ListPrimary(), hwCores.ListSecondary()...)
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
			availBySocket[socket] = socketCores
		}
	}
	return availBySocket
}
