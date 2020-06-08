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

var ErrEalInitialized = errors.New("EAL is already initialized")

// Initialize DPDK Environment Abstraction Layer (EAL).
func InitEal(args []string) (remainingArgs []string, e error) {
	if isEalInitialized {
		return nil, ErrEalInitialized
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

// Initialize EAL, and panic if it fails.
func MustInitEal(args []string) (remainingArgs []string) {
	var e error
	remainingArgs, e = InitEal(args)
	if e != nil && e != ErrEalInitialized {
		panic(e)
	}
	return remainingArgs
}

// Indicate a NUMA socket.
type NumaSocket int

const NUMA_SOCKET_ANY = NumaSocket(C.SOCKET_ID_ANY)

func (socket NumaSocket) Match(other NumaSocket) bool {
	return socket == NUMA_SOCKET_ANY || other == NUMA_SOCKET_ANY || socket == other
}

func (socket NumaSocket) String() string {
	if socket == NUMA_SOCKET_ANY {
		return "any"
	}
	return fmt.Sprintf("%d", socket)
}

// Indicate state of LCore.
type LCoreState int

const (
	LCORE_STATE_WAIT LCoreState = iota
	LCORE_STATE_RUNNING
	LCORE_STATE_FINISHED
)

func (s LCoreState) String() string {
	switch s {
	case LCORE_STATE_WAIT:
		return "WAIT"
	case LCORE_STATE_RUNNING:
		return "RUNNING"
	case LCORE_STATE_FINISHED:
		return "FINISHED"
	}
	return fmt.Sprintf("LCoreState(%d)", s)
}

// A logical core.
type LCore uint

const LCORE_INVALID = LCore(C.LCORE_ID_ANY)

func (lc LCore) IsValid() bool {
	return lc != LCORE_INVALID
}

func (lc LCore) IsMaster() bool {
	return lc == GetMasterLCore()
}

func (lc LCore) String() string {
	if !lc.IsValid() {
		return "(invalid)"
	}
	return strconv.Itoa(int(lc))
}

func (lc LCore) GetNumaSocket() NumaSocket {
	if !lc.IsValid() {
		return NUMA_SOCKET_ANY
	}
	return NumaSocket(C.rte_lcore_to_socket_id(C.uint(lc)))
}

func (lc LCore) GetState() LCoreState {
	panicInSlave("LCore.GetState()")
	return LCoreState(C.rte_eal_get_lcore_state(C.uint(lc)))
}

// Asynchronously launch a function on an lcore.
// Returns whether success.
func (lc LCore) RemoteLaunch(f func() int) bool {
	panicInSlave("LCore.RemoteLaunch()")
	lcoreFuncs[lc] = f
	res := C.rte_eal_remote_launch((*C.lcore_function_t)(C.go_lcoreLaunch), nil, C.uint(lc))
	return res == 0
}

// Wait for lcore to finish running, and return lcore function's return value.
// If lcore is not running, return 0 immediately.
func (lc LCore) Wait() int {
	panicInSlave("LCore.Wait()")
	return int(C.rte_eal_wait_lcore(C.uint(lc)))
}

var lcoreFuncs [C.RTE_MAX_LCORE]func() int

//export go_lcoreLaunch
func go_lcoreLaunch(ctx unsafe.Pointer) C.int {
	return C.int(lcoreFuncs[C.rte_lcore_id()]())
}

// Prevent a function to be executed in slave lcore.
func panicInSlave(funcName string) {
	lc := GetCurrentLCore()
	if lc.IsValid() && !lc.IsMaster() {
		panic(fmt.Sprintf("%s is unavailable in slave lcore; current=%d master=%d",
			funcName, lc, GetMasterLCore()))
	}
	// 'invalid' lcore is permitted, because Golang runtime could use another thread
}

func GetCurrentLCore() LCore {
	return LCore(C.rte_lcore_id())
}

func GetMasterLCore() LCore {
	return LCore(C.rte_get_master_lcore())
}

func ListSlaveLCores() (list []LCore) {
	for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
		list = append(list, LCore(i))
	}
	return list
}

func ListNumaSocketsOfLCores(lcores []LCore) (a []NumaSocket) {
	a = make([]NumaSocket, len(lcores))
	for i, lcore := range lcores {
		a[i] = lcore.GetNumaSocket()
	}
	return a
}
