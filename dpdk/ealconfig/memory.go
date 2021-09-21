package ealconfig

import (
	"strconv"

	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
)

// MemoryConfig contains memory related configuration.
type MemoryConfig struct {
	// MemChannels is the number of memory channels.
	// Omitting or setting an incorrect value may result in suboptimal performance.
	MemChannels int `json:"memChannels,omitempty"`

	// MemPerNuma maps from NUMA socket ID to the amount of memory (MiB).
	// Hugepages must be configured prior to starting NDN-DPDK.
	//
	// Example:
	//  MemPerNuma[0] = 16384  limits up to 16384MB on socket 0.
	//  Omitting MemPerNuma[1] places no memory limit on socket 1.
	//  MemPerNuma[2] = 0      limits up to 1MB on socket 2; DPDK does not support a zero limit.
	MemPerNuma map[int]int `json:"memPerNuma,omitempty"`

	// PreallocateMem preallocates memory up to the limit on each NUMA socket.
	// If a NUMA socket has no limit (MemPerNuma[socket] is omitted), this preallocates 1MB.
	PreallocateMem bool `json:"preallocateMem,omitempty"`

	// FilePrefix is shared data file prefix.
	// Each independent instance of NDN-DPDK must have different FilePrefix.
	FilePrefix string `json:"filePrefix,omitempty"`

	// MemFlags is memory-related flags passed to DPDK.
	// This replaces all other options.
	MemFlags string `json:"memFlags,omitempty"`
}

func (cfg MemoryConfig) args(hwInfo hwinfo.Provider) (args []string, e error) {
	if cfg.MemFlags != "" {
		return shellSplit("MemFlags", cfg.MemFlags)
	}

	if cfg.MemChannels > 0 {
		args = append(args, "-n", strconv.Itoa(cfg.MemChannels))
	}

	if len(cfg.MemPerNuma) > 0 {
		var socketMem, socketLimit commaSeparated
		for socket, maxSocket := 0, hwInfo.Cores().MaxNumaSocket(); socket <= maxSocket; socket++ {
			limit, hasLimit := cfg.MemPerNuma[socket]
			switch {
			case !hasLimit:
				socketMem.AppendInt(1)
				socketLimit.AppendInt(0)
			case limit <= 0:
				socketMem.AppendInt(0)
				socketLimit.AppendInt(1)
			default:
				socketMem.AppendInt(limit)
				socketLimit.AppendInt(limit)
			}
		}
		args = append(args, "--socket-limit", socketLimit.String())
		if cfg.PreallocateMem {
			args = append(args, "--socket-mem", socketMem.String())
		}
	}

	if cfg.FilePrefix != "" {
		args = append(args, "--file-prefix", cfg.FilePrefix)
	}

	args = append(args, "--in-memory", "--single-file-segments")
	return args, nil
}
