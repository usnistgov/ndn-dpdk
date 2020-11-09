package ealconfig

import (
	"strconv"
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
	//  MemPerNuma[1] = 0      limits up to 1MB on socket 2; DPDK does not support a zero limit.
	MemPerNuma map[int]int `json:"memPerNuma,omitempty"`

	// FilePrefix is shared data file prefix.
	// Each independent instance of NDN-DPDK must have different FilePrefix.
	FilePrefix string `json:"filePrefix,omitempty"`

	// DisableHugeUnlink disables unlink hugepage map files.
	DisableHugeUnlink bool `json:"disableHugeUnlink,omitempty"`

	// MemFlags is memory-related flags passed to DPDK.
	// This replaces all other options.
	MemFlags string `json:"memFlags,omitempty"`
}

func (cfg MemoryConfig) args(req Request, hwInfo HwInfoSource) (args []string, e error) {
	if cfg.MemFlags != "" {
		return shellSplit("MemFlags", cfg.MemFlags)
	}

	if cfg.MemChannels > 0 {
		args = append(args, "-n", strconv.Itoa(cfg.MemChannels))
	}

	if len(cfg.MemPerNuma) > 0 {
		var socketLimit commaSeparatedNumbers
		for socket, maxSocket := 0, maxNumaSocket(hwInfo); socket <= maxSocket; socket++ {
			limit, hasLimit := cfg.MemPerNuma[socket]
			switch {
			case !hasLimit:
				limit = 0
			case limit <= 0:
				limit = 1
			}
			socketLimit.Append(limit)
		}
		args = append(args, "--socket-limit", socketLimit.String())
	}

	if cfg.FilePrefix != "" {
		args = append(args, "--file-prefix", cfg.FilePrefix)
	}

	if !cfg.DisableHugeUnlink {
		args = append(args, "--huge-unlink")
	}

	return args, nil
}
