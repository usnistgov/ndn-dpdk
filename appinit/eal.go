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
	lc := NewLCoreReservations().Reserve(socket)
	ok := lc.RemoteLaunch(f)
	if !ok {
		return dpdk.LCORE_INVALID
	}
	return lc
}

// Asynchonrously launch a function on an lcore in specified NumaSocket.
// os.Exit(EXIT_EAL_LAUNCH_ERROR) if no lcore available or other failure.
func LaunchRequired(f dpdk.LCoreFunc, socket dpdk.NumaSocket) dpdk.LCore {
	lc := NewLCoreReservations().Reserve(socket)
	ok := lc.RemoteLaunch(f)
	if !ok {
		log.Printf("unable to launch lcore %d on socket %d", lc, socket)
		os.Exit(EXIT_EAL_LAUNCH_ERROR)
	}
	return lc
}

// Reserve lcores for launching later.
// The same LCoreReservations instance must be for reserving multiple lcores,
// and no other launching is allowed during reservation process.
type LCoreReservations map[dpdk.LCore]bool

func NewLCoreReservations() LCoreReservations {
	return make(LCoreReservations)
}

// Reserve an idle lcore in specified NumaSocket.
// Return dpdk.LCORE_INVALID if no lcore available.
func (lcr LCoreReservations) Reserve(socket dpdk.NumaSocket) dpdk.LCore {
	for _, lc := range Eal.Slaves {
		if lcr[lc] || lc.GetState() != dpdk.LCORE_STATE_WAIT ||
			(socket != dpdk.NUMA_SOCKET_ANY && lc.GetNumaSocket() != socket) {
			continue
		}
		lcr[lc] = true
		return lc
	}
	return dpdk.LCORE_INVALID
}

// Reserve an idle lcore in specified NumaSocket.
// os.Exit(EXIT_EAL_LAUNCH_ERROR) if no lcore available.
func (lcr LCoreReservations) ReserveRequired(socket dpdk.NumaSocket) dpdk.LCore {
	lc := lcr.Reserve(socket)
	if !lc.IsValid() {
		log.Printf("unable to reserve an lcore on socket %d", socket)
		os.Exit(EXIT_EAL_LAUNCH_ERROR)
	}
	return lc
}
