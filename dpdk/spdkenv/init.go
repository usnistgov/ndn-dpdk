// Package spdkenv contains bindings of SPDK environment and threads.
package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/env_dpdk.h>
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/dlopen"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var mainThread *Thread

// InitEnv initializes the SPDK environment.
// Errors are fatal.
func InitEnv() {
	// As of SPDK 20.10, libspdk_event.so depends on rte_power_set_freq symbol exported by
	// librte_power.so but does not link with that library.
	dlopen.Load("/usr/local/lib/librte_power.so")

	e := dlopen.LoadGroup("/usr/local/lib/libspdk.so")
	if e != nil {
		log.Fatalf("SPDK dlopen error %s", e)
		return
	}

	if res := int(C.spdk_env_dpdk_post_init(C.bool(false))); res != 0 {
		log.Fatalf("SPDK env init error %s", eal.Errno(-res))
		return
	}

	initLogging()
}

// InitMainThread creates a main thread, and launches on the current goroutine.
// This should be invoked on the MainLCore.
// This function never returns.
func InitMainThread(assignThread chan<- *Thread) {
	if lc := eal.CurrentLCore(); lc != eal.MainLCore {
		log.Panicf("lc=%v is not main=%v", lc, eal.MainLCore)
	}

	var e error
	mainThread, e = NewThread()
	if e != nil {
		log.Fatalf("SPDK thread error %s", e)
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
		log.Fatalf("SPDK RPC init error %s", e)
		return
	}
}
