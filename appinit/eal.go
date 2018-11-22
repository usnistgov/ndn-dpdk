package appinit

import (
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
		log.WithError(e).Fatal("EAL init failed")
	}
}

// Asynchronously launch a function on an lcore in specified NumaSocket.
// Return the lcore used, or dpdk.LCORE_INVALID if no lcore available or other failure.
func Launch(f dpdk.LCoreFunc, socket dpdk.NumaSocket) dpdk.LCore {
	lc := NewLCoreReservations().Reserve(socket)
	ok := lc.RemoteLaunch(f)
	if !ok {
		return dpdk.LCORE_INVALID
	}
	return lc
}

// Asynchronously launch a function on an lcore in specified NumaSocket.
// Fatal error if no lcore available or other failure.
func MustLaunch(f dpdk.LCoreFunc, socket dpdk.NumaSocket) dpdk.LCore {
	lc := NewLCoreReservations().Reserve(socket)
	ok := lc.RemoteLaunch(f)
	if !ok {
		log.WithFields(makeLogFields("lcore", lc, "socket", socket)).Fatal("lcore launch failed")
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

// Indicate lcores are reserved.
func (lcr LCoreReservations) MarkReserved(lcores ...dpdk.LCore) {
	for _, lc := range lcores {
		lcr[lc] = true
	}
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
// Fatal error if no lcore available.
func (lcr LCoreReservations) MustReserve(socket dpdk.NumaSocket) dpdk.LCore {
	lc := lcr.Reserve(socket)
	if !lc.IsValid() {
		log.WithFields(makeLogFields("socket", socket)).Fatal("no lcore available")
	}
	return lc
}

// Launch a thread.
// Fatal error if no lcore available or launch error.
func MustLaunchThread(thread dpdk.IThread, socket dpdk.NumaSocket) {
	lc := NewLCoreReservations().MustReserve(socket)
	thread.SetLCore(lc)
	if e := thread.Launch(); e != nil {
		log.WithError(e).Fatal("thread launch failed")
	}
}
