package appinit

import (
	"ndn-dpdk/dpdk"
)

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
	for _, lc := range dpdk.ListSlaveLCores() {
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
