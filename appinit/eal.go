package appinit

import (
	"log"
	"os"

	"ndn-dpdk/dpdk"
)

var Eal *dpdk.Eal

func InitEal() {
	if Eal != nil {
		return
	}

	var e error
	Eal, e = dpdk.NewEal(os.Args)
	if e != nil {
		Exitf(EXIT_EAL_INIT_ERROR, "NewEal(): %v", e)
	}
}

// Asynchonrously launch a function on an lcore in specified NumaSocket.
// Return the lcore used, or dpdk.LCORE_INVALID if no lcore available or other failure.
func Launch(f dpdk.LCoreFunc, socket dpdk.NumaSocket) dpdk.LCore {
	for _, lcore := range Eal.Slaves {
		if lcore.GetState() != dpdk.LCORE_STATE_WAIT {
			continue
		}
		if socket != dpdk.NUMA_SOCKET_ANY && lcore.GetNumaSocket() != socket {
			continue
		}
		if !lcore.RemoteLaunch(f) {
			break
		}
		return lcore
	}
	return dpdk.LCORE_INVALID
}

// Asynchonrously launch a function on an lcore in specified NumaSocket.
// os.Exit(EXIT_EAL_LAUNCH_ERROR) if no lcore available or other failure.
func LaunchRequired(f dpdk.LCoreFunc, socket dpdk.NumaSocket) dpdk.LCore {
	lcore := Launch(f, socket)
	if !lcore.IsValid() {
		log.Printf("unable to launch lcore on socket %d", socket)
		for _, lcore := range Eal.Slaves {
			log.Printf("lcore %d (socket %d) is %v", lcore, lcore.GetNumaSocket(), lcore.GetState())
		}
		os.Exit(EXIT_EAL_LAUNCH_ERROR)
	}
	return lcore
}
