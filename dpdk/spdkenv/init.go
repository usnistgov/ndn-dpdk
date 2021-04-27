// Package spdkenv contains bindings of SPDK environment and threads.
package spdkenv

/*
#include "../../csrc/core/logger.h"
#include <spdk/env_dpdk.h>
#include <spdk/log.h>

void c_SpdkLoggerReady()
{
	SPDK_NOTICELOG("SPDK logger ready\n");
}
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/dlopen"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var logger = logging.New("spdkenv")

var mainThread *Thread

// InitEnv initializes the SPDK environment.
// Errors are fatal.
func InitEnv() {
	// As of SPDK 20.10, libspdk_event.so depends on rte_power_set_freq symbol exported by
	// librte_power.so but does not link with that library.
	dlopen.Load("/usr/local/lib/librte_power.so")

	e := dlopen.LoadGroup("/usr/local/lib/libspdk.so")
	if e != nil {
		logger.Fatal("SPDK dlopen error", zap.Error(e))
		return
	}

	C.spdk_log_open((*C.logfunc)(C.Logger_Spdk))

	if res := int(C.spdk_env_dpdk_post_init(C.bool(false))); res != 0 {
		logger.Fatal("SPDK env init error",
			zap.Error(eal.Errno(-res)),
		)
		return
	}

	C.c_SpdkLoggerReady()
}

// InitMainThread creates a main thread, and launches on the current goroutine.
// This should be invoked on the MainLCore.
// This function never returns.
func InitMainThread(assignThread chan<- *Thread) {
	if lc := eal.CurrentLCore(); lc != eal.MainLCore {
		logger.Panic("lcore is not main",
			lc.ZapField("lc"),
			eal.MainLCore.ZapField("main"),
		)
	}

	var e error
	mainThread, e = NewThread()
	if e != nil {
		logger.Fatal("SPDK thread error", zap.Error(e))
		return
	}
	mainThread.SetLCore(eal.MainLCore)
	assignThread <- mainThread
	mainThread.main()
}

// InitFinal finishes initializing SPDK.
// Errors are fatal.
func InitFinal() {
	if e := initRPC(); e != nil {
		logger.Fatal("SPDK RPC init error", zap.Error(e))
		return
	}
}
