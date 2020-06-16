package eal

/*
#include "../../csrc/core/common.h"

#include <rte_eal.h>
#include <rte_lcore.h>
#include <rte_random.h>

extern int go_lcoreLaunch(void*);
*/
import "C"
import (
	"math/rand"
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

var ealInitOnce sync.Once

// InitEal initializes the DPDK Environment Abstraction Layer (EAL).
// Errors are fatal.
func InitEal(args []string) (remainingArgs []string) {
	ealInitOnce.Do(func() {
		a := cptr.NewCArgs(args)
		defer a.Close()

		res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
		if res < 0 {
			log.Fatalf("EAL init error %s", GetErrno())
			return
		}

		rand.Seed(int64(C.rte_rand()))
		remainingArgs = a.GetRemainingArgs(int(res))
	})
	return remainingArgs
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
