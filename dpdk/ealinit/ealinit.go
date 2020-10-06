package ealinit

/*
#include "../../csrc/core/common.h"

#include <rte_eal.h>
#include <rte_lcore.h>
#include <rte_random.h>
*/
import "C"
import (
	"os"
	"runtime"
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

var initOnce sync.Once

// Init initializes DPDK and SPDK.
// args should not include program name.
// Panics on error.
func Init(args []string) {
	initOnce.Do(func() {
		assignMainThread := make(chan *spdkenv.Thread)
		go func() {
			runtime.LockOSThread()
			initEal(args)
			spdkenv.InitEnv()
			spdkenv.InitMainThread(assignMainThread) // never returns
		}()
		th := <-assignMainThread
		eal.MainThread = th
		eal.MainReadSide = th.RcuReadSide
		eal.CallMain(func() {
			log.Debug("MainThread is running")
		})
		spdkenv.InitFinal()
	})
	return
}

func initEal(args []string) {
	logEntry := log.WithField("args", args)
	exe, e := os.Executable()
	if e != nil {
		exe = os.Args[0]
	}
	argv := append([]string{exe}, args...)
	a := cptr.NewCArgs(argv)
	defer a.Close()

	res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
	if res < 0 {
		logEntry.Fatalf("EAL init error %v", eal.GetErrno())
		return
	}

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
	logEntry.WithFields(makeLogFields("initial", eal.Initial, "workers", eal.Workers, "sockets", eal.Sockets)).Info("EAL ready")
}
