package dpdk

/*
extern int go_lcoreLaunch(void*);

#include <dlfcn.h> // dlopen()
#include <rte_config.h>
#include <rte_eal.h>
#include <rte_launch.h>
#include <rte_lcore.h>
#include <rte_memory.h>
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"strings"
	"unsafe"
)

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

type LCore uint

const LCORE_INVALID = LCore(C.LCORE_ID_ANY)

func GetCurrentLCore() LCore {
	return LCore(C.rte_lcore_id())
}

func GetMasterLCore() LCore {
	return LCore(C.rte_get_master_lcore())
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

func (lc LCore) IsValid() bool {
	return lc != LCORE_INVALID
}

func (lc LCore) IsMaster() bool {
	return lc == GetMasterLCore()
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

func ListNumaSocketsOfLCores(lcores []LCore) (a []NumaSocket) {
	a = make([]NumaSocket, len(lcores))
	for i, lcore := range lcores {
		a[i] = lcore.GetNumaSocket()
	}
	return a
}

type LCoreFunc func() int

var lcoreFuncs = make([]LCoreFunc, int(C.RTE_MAX_LCORE))

//export go_lcoreLaunch
func go_lcoreLaunch(lc unsafe.Pointer) C.int {
	return C.int(lcoreFuncs[uintptr(lc)]())
}

// Asynchronously launch a function on an lcore.
// Returns whether success.
func (lc LCore) RemoteLaunch(f LCoreFunc) bool {
	panicInSlave("LCore.RemoteLaunch()")
	lcoreFuncs[lc] = f
	res := C.rte_eal_remote_launch((*C.lcore_function_t)(C.go_lcoreLaunch),
		unsafe.Pointer(uintptr(lc)), C.uint(lc))
	return res == 0
}

// Wait for lcore to finish running, and return lcore function's return value.
// If lcore is not running, return 0 immediately.
func (lc LCore) Wait() int {
	panicInSlave("LCore.Wait()")
	return int(C.rte_eal_wait_lcore(C.uint(lc)))
}

type Eal struct {
	Args   []string // remaining command-line arguments
	Master LCore
	Slaves []LCore
}

// Initialize DPDK Environment Abstraction Layer (EAL).
func NewEal(args []string) (*Eal, error) {
	e := loadDpdkDynLibs()
	if e != nil {
		return nil, e
	}
	eal := new(Eal)

	a := NewCArgs(args)
	defer a.Close()

	res := int(C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv)))
	if res < 0 {
		return nil, GetErrno()
	}
	eal.Args = a.GetRemainingArgs(res)

	eal.Master = LCore(C.rte_get_master_lcore())

	for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
		eal.Slaves = append(eal.Slaves, LCore(i))
	}

	return eal, nil
}

func loadDpdkDynLibs() (e error) {
	var libdpdkPath string
	libdpdkPaths := []string{
		"/usr/local/lib/libdpdk.so",
		"/usr/lib/x86_64-linux-gnu/libdpdk.so",
	}
	var dpdkSoContent []byte
	for _, libdpdkPath = range libdpdkPaths {
		dpdkSoContent, e = ioutil.ReadFile(libdpdkPath)
		if e == nil {
			break
		}
	}
	if e != nil {
		return e
	}

	dpdkText := strings.Split(string(dpdkSoContent), " ")
	if len(dpdkText) < 4 || dpdkText[0] != "GROUP" {
		return fmt.Errorf("unexpected text in %s", libdpdkPath)
	}

	for _, soname := range dpdkText[2 : len(dpdkText)-1] {
		cSoname := C.CString(soname)
		defer C.free(unsafe.Pointer(cSoname))
		h := C.dlopen(cSoname, C.RTLD_LAZY)
		if h == nil {
			return fmt.Errorf("dlopen failed for %s", soname)
		}
	}

	return nil
}
