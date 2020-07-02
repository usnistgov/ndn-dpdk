package eal

/*
#include "../../csrc/core/common.h"

#include <rte_launch.h>
#include <rte_lcore.h>

extern int go_lcoreLaunch(void*);
*/
import "C"
import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// MaxLCoreID is the maximum LCore ID.
const MaxLCoreID = C.RTE_MAX_LCORE

// LCore represents a logical core.
// Zero LCore is invalid.
type LCore struct {
	v int // lcore ID + 1
}

// LCoreFromID converts lcore ID to LCore.
func LCoreFromID(id int) (lc LCore) {
	if id < 0 || id > C.RTE_MAX_LCORE {
		return lc
	}
	lc.v = id + 1
	return lc
}

// ID returns lcore ID.
func (lc LCore) ID() int {
	return lc.v - 1
}

// Valid returns true if this is a valid lcore (not zero value).
func (lc LCore) Valid() bool {
	return lc.v != 0
}

func (lc LCore) String() string {
	if !lc.Valid() {
		return "invalid"
	}
	return strconv.Itoa(lc.ID())
}

// NumaSocket returns the NUMA socket where this lcore is located.
func (lc LCore) NumaSocket() (socket NumaSocket) {
	if !lc.Valid() {
		return socket
	}
	return NumaSocketFromID(int(C.rte_lcore_to_socket_id(C.uint(lc.ID()))))
}

// IsBusy returns true if this lcore is running a function.
func (lc LCore) IsBusy() bool {
	panicInWorker("LCore.IsBusy()")
	return C.rte_eal_get_lcore_state(C.uint(lc.ID())) != C.WAIT
}

// RemoteLaunch asynchronously launches a function on this lcore.
func (lc LCore) RemoteLaunch(f func() int) error {
	panicInWorker("LCore.RemoteLaunch()")
	if !lc.Valid() {
		panic("invalid lcore")
	}
	ctx := cptr.CtxPut(f)
	res := C.rte_eal_remote_launch((*C.lcore_function_t)(C.go_lcoreLaunch), ctx, C.uint(lc.ID()))
	if res != 0 {
		return Errno(-res)
	}
	return nil
}

// Wait blocks until this lcore finishes running, and returns lcore function's return value.
// If this lcore is not running, returns 0 immediately.
func (lc LCore) Wait() int {
	panicInWorker("LCore.Wait()")
	return int(C.rte_eal_wait_lcore(C.uint(lc.ID())))
}

//export go_lcoreLaunch
func go_lcoreLaunch(ctx unsafe.Pointer) C.int {
	f := cptr.CtxPop(ctx).(func() int)
	return C.int(f())
}

// Prevent a function from executing in worker lcore.
func panicInWorker(funcName string) {
	lc := GetCurrentLCore()
	if initial := GetInitialLCore(); lc.Valid() && lc.ID() != initial.ID() {
		panic(fmt.Sprintf("%s is unavailable in worker lcore; current=%s initial=%s",
			funcName, lc, initial))
	}
	// 'invalid' lcore is permitted, because Go runtime could use another thread
}

// ListNumaSocketsOfLCores maps lcores into NUMA sockets.
func ListNumaSocketsOfLCores(lcores []LCore) (a []NumaSocket) {
	a = make([]NumaSocket, len(lcores))
	for i, lcore := range lcores {
		a[i] = lcore.NumaSocket()
	}
	return a
}
