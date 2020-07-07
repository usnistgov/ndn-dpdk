package ealinit

/*
#include "../../csrc/core/common.h"

#include <rte_eal.h>
#include <rte_lcore.h>
#include <rte_random.h>
*/
import "C"
import (
	"math/rand"
	"runtime"
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

var initOnce sync.Once

// Init initializes DPDK and SPDK.
func Init(args []string) (remainingArgs []string) {
	initOnce.Do(func() {
		assignMainThread := make(chan eal.PollThread)
		go func() {
			runtime.LockOSThread()
			remainingArgs = initEal(args)
			spdkenv.InitEnv()
			spdkenv.InitMainThread(assignMainThread) // never returns
		}()
		eal.MainThread = <-assignMainThread
		eal.CallMain(func() {
			log.Info("MainThread is running")
		})
		spdkenv.InitFinal()
	})
	return
}

func initEal(args []string) (remainingArgs []string) {
	a := cptr.NewCArgs(args)
	defer a.Close()

	res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
	if res < 0 {
		log.Fatalf("EAL init error %v", eal.GetErrno())
		return
	}

	rand.Seed(int64(C.rte_rand()))
	remainingArgs = a.RemainingArgs(int(res))

	eal.Initial = eal.LCoreFromID(int(C.rte_get_master_lcore()))
	hasSocket := make(map[eal.NumaSocket]bool)
	for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
		lc := eal.LCoreFromID(int(i))
		eal.Workers = append(eal.Workers, lc)
		if socket := lc.NumaSocket(); !hasSocket[socket] {
			eal.Sockets = append(eal.Sockets, socket)
			hasSocket[socket] = true
		}
	}
	log.WithFields(makeLogFields("initial", eal.Initial, "workers", eal.Workers, "sockets", eal.Sockets)).Info("EAL ready")
	return remainingArgs
}
