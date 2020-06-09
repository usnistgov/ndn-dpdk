package dpdk

/*
#include "../core/common.h"

#include <rte_eal.h>
#include <rte_launch.h>
#include <rte_lcore.h>
#include <rte_random.h>

extern int go_lcoreLaunch(void*);
*/
import "C"
import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"unsafe"
)

var isEalInitialized = false

var errEalInitialized = errors.New("EAL is already initialized")

// InitEal initializes the DPDK Environment Abstraction Layer (EAL).
func InitEal(args []string) (remainingArgs []string, e error) {
	if isEalInitialized {
		return nil, errEalInitialized
	}

	a := NewCArgs(args)
	defer a.Close()

	res := int(C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv)))
	if res < 0 {
		return nil, GetErrno()
	}

	rand.Seed(int64(C.rte_rand()))

	isEalInitialized = true
	return a.GetRemainingArgs(res), nil
}

// MustInitEal initializes EAL, and panics if it fails.
func MustInitEal(args []string) (remainingArgs []string) {
	var e error
	remainingArgs, e = InitEal(args)
	if e != nil && e != errEalInitialized {
		panic(e)
	}
	return remainingArgs
}

// NumaSocket represents a NUMA socket.
// Zero value is SOCKET_ID_ANY.
type NumaSocket struct {
	v int // socket ID + 1
}

// NumaSocketFromID converts socket ID to NumaSocket.
func NumaSocketFromID(id int) (socket NumaSocket) {
	if id < 0 || id > C.RTE_MAX_NUMA_NODES {
		return socket
	}
	socket.v = id + 1
	return socket
}

// ID returns NUMA socket ID.
func (socket NumaSocket) ID() int {
	return socket.v - 1
}

// IsAny returns true if this represents SOCKET_ID_ANY.
func (socket NumaSocket) IsAny() bool {
	return socket.v == 0
}

func (socket NumaSocket) String() string {
	if socket.IsAny() {
		return "any"
	}
	return strconv.Itoa(socket.ID())
}

// LCore represents a logical core.
// Zero value is invalid lcore.
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

// IsValid returns true if this is a valid lcore (not zero value).
func (lc LCore) IsValid() bool {
	return lc.v != 0
}

func (lc LCore) String() string {
	if !lc.IsValid() {
		return "(invalid)"
	}
	return strconv.Itoa(int(lc.ID()))
}

// GetNumaSocket returns the NUMA socket where this lcore is located.
func (lc LCore) GetNumaSocket() (socket NumaSocket) {
	if !lc.IsValid() {
		return socket
	}
	return NumaSocketFromID(int(C.rte_lcore_to_socket_id(C.uint(lc.ID()))))
}

// IsBusy returns true if this lcore is running a function.
func (lc LCore) IsBusy() bool {
	panicInSlave("LCore.GetState()")
	return C.rte_eal_get_lcore_state(C.uint(lc.ID())) != C.WAIT
}

// RemoteLaunch asynchronously launches a function on this lcore.
// Returns whether success.
func (lc LCore) RemoteLaunch(f func() int) bool {
	panicInSlave("LCore.RemoteLaunch()")
	if !lc.IsValid() {
		panic("invalid lcore")
	}
	lcoreFuncs[lc.ID()] = f
	res := C.rte_eal_remote_launch((*C.lcore_function_t)(C.go_lcoreLaunch), nil, C.uint(lc.ID()))
	return res == 0
}

// Wait blocks until this lcore finishes running, and returns lcore function's return value.
// If this lcore is not running, returns 0 immediately.
func (lc LCore) Wait() int {
	panicInSlave("LCore.Wait()")
	return int(C.rte_eal_wait_lcore(C.uint(lc.ID())))
}

var lcoreFuncs [C.RTE_MAX_LCORE]func() int

//export go_lcoreLaunch
func go_lcoreLaunch(ctx unsafe.Pointer) C.int {
	return C.int(lcoreFuncs[C.rte_lcore_id()]())
}

// Prevent a function from executing in slave lcore.
func panicInSlave(funcName string) {
	lc := GetCurrentLCore()
	if master := GetMasterLCore(); lc.IsValid() && lc.ID() != master.ID() {
		panic(fmt.Sprintf("%s is unavailable in slave lcore; current=%s master=%s",
			funcName, lc, master))
	}
	// 'invalid' lcore is permitted, because Go runtime could use another thread
}

// GetCurrentLCore returns the current lcore.
func GetCurrentLCore() LCore {
	return LCoreFromID(int(C.rte_lcore_id()))
}

// GetMasterLCore returns the master lcore.
func GetMasterLCore() LCore {
	return LCoreFromID(int(C.rte_get_master_lcore()))
}

// ListSlaveLCores returns a list of slave lcores.
func ListSlaveLCores() (list []LCore) {
	for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
		list = append(list, LCoreFromID(int(i)))
	}
	return list
}

// ListNumaSocketsOfLCores maps lcores into NUMA sockets.
func ListNumaSocketsOfLCores(lcores []LCore) (a []NumaSocket) {
	a = make([]NumaSocket, len(lcores))
	for i, lcore := range lcores {
		a[i] = lcore.GetNumaSocket()
	}
	return a
}
