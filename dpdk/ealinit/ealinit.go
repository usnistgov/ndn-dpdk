// Package ealinit initializes DPDK EAL and SPDK main thread.
package ealinit

/*
#include "../../csrc/dpdk/mbuf.h"
#include <rte_eal.h>
#include <rte_lcore.h>
#include <rte_random.h>
*/
import "C"
import (
	"os"
	"runtime"
	"sync"

	"github.com/kballard/go-shellquote"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

func init() {
	ealconfig.PmdPath = C.RTE_EAL_PMD_PATH
}

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
			initMbufDynfields()
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
	logEntry := log.WithField("args", shellquote.Join(args...))
	exe, e := os.Executable()
	if e != nil {
		exe = os.Args[0]
	}
	argv := append([]string{exe}, args...)
	a := cptr.NewCArgs(argv)
	defer a.Close()

	C.rte_mp_disable()
	res := C.rte_eal_init(C.int(a.Argc), (**C.char)(a.Argv))
	if res < 0 {
		logEntry.WithError(eal.GetErrno()).Fatal("EAL init error")
		return
	}

	eal.MainLCore = eal.LCoreFromID(int(C.rte_get_main_lcore()))
	hasSocket := make(map[eal.NumaSocket]bool)
	for i := C.rte_get_next_lcore(C.RTE_MAX_LCORE, 1, 1); i < C.RTE_MAX_LCORE; i = C.rte_get_next_lcore(i, 1, 0) {
		lc := eal.LCoreFromID(int(i))
		eal.Workers = append(eal.Workers, lc)
		if socket := lc.NumaSocket(); !hasSocket[socket] {
			eal.Sockets = append(eal.Sockets, socket)
			hasSocket[socket] = true
		}
	}
	logEntry.WithFields(makeLogFields("main", eal.MainLCore, "workers", eal.Workers, "sockets", eal.Sockets)).Info("EAL ready")
}

func initMbufDynfields() {
	ok := bool(C.Mbuf_RegisterDynFields())
	if !ok {
		log.WithError(eal.GetErrno()).Fatal("mbuf dynfields init error")
	}
}
