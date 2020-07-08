package eal

/*
#include "../../csrc/core/common.h"

#include <rte_launch.h>
#include <rte_lcore.h>
*/
import "C"
import (
	"strconv"

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

// CurrentLCore returns the current lcore.
func CurrentLCore() LCore {
	return LCoreFromID(int(C.rte_lcore_id()))
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
	return CallMain(func() bool {
		return C.rte_eal_get_lcore_state(C.uint(lc.ID())) != C.WAIT
	}).(bool)
}

// RemoteLaunch asynchronously launches a function on this lcore.
func (lc LCore) RemoteLaunch(fn cptr.Function) error {
	if !lc.Valid() {
		panic("invalid lcore")
	}
	res := CallMain(func() C.int {
		f, arg := cptr.Func0.CallbackOnce(fn)
		return C.rte_eal_remote_launch((*C.lcore_function_t)(f), arg, C.uint(lc.ID()))
	}).(C.int)
	if res != 0 {
		return Errno(-res)
	}
	return nil
}

// Wait blocks until this lcore finishes running, and returns lcore function's return value.
// If this lcore is not running, returns 0 immediately.
func (lc LCore) Wait() int {
	return CallMain(func() int {
		return int(C.rte_eal_wait_lcore(C.uint(lc.ID())))
	}).(int)
}
