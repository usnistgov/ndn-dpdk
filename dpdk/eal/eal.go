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

// LCore and NUMA sockets, available after Init().
var (
	Initial LCore
	Workers []LCore
	Sockets []NumaSocket

	ealInitOnce sync.Once
)

// Init initializes the DPDK Environment Abstraction Layer (EAL).
// Errors are fatal.
func Init(args []string) (remainingArgs []string) {
	ealInitOnce.Do(func() {
		a := cptr.NewCArgs(args)
		defer a.Close()

		res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
		if res < 0 {
			log.Fatalf("EAL init error %v", GetErrno())
			return
		}

		rand.Seed(int64(C.rte_rand()))
		remainingArgs = a.RemainingArgs(int(res))

		Initial = LCoreFromID(int(C.rte_get_master_lcore()))
		hasSocket := make(map[NumaSocket]bool)
		for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
			lc := LCoreFromID(int(i))
			Workers = append(Workers, lc)
			if socket := lc.NumaSocket(); !hasSocket[socket] {
				Sockets = append(Sockets, socket)
				hasSocket[socket] = true
			}
		}
		log.WithFields(makeLogFields("initial", Initial, "workers", Workers, "sockets", Sockets)).Info("EAL ready")
	})
	return remainingArgs
}
