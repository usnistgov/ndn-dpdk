// Package spdkenv contains bindings of SPDK environment and threads.
package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/env_dpdk.h>
#include <spdk/log.h>
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
		logger.Fatal("SPDK dlopen error",
			zap.Error(e),
		)
		return
	}

	if res := int(C.spdk_env_dpdk_post_init(C.bool(false))); res != 0 {
		logger.Fatal("SPDK env init error",
			zap.Error(eal.Errno(-res)),
		)
		return
	}

	C.spdk_log_set_print_level(func() C.enum_spdk_log_level {
		switch logging.GetLevel("SPDK") {
		case 'V':
			return C.SPDK_LOG_DEBUG
		case 'D':
			return C.SPDK_LOG_INFO
		case 'I':
			return C.SPDK_LOG_NOTICE
		case 'W':
			return C.SPDK_LOG_WARN
		case 'E', 'F':
			return C.SPDK_LOG_ERROR
		case 'N':
			return C.SPDK_LOG_DISABLED
		}
		return C.SPDK_LOG_INFO
	}())

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
		logger.Fatal("SPDK thread error",
			zap.Error(e),
		)
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
		logger.Fatal("SPDK RPC init error",
			zap.Error(e),
		)
		return
	}
}
