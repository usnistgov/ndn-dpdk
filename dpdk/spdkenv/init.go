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
	e := dlopen.LoadDynLibs("/usr/local/lib/libspdk.so")
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
// This should be invoked on the Initial lcore.
// This function never returns.
func InitMainThread(assignThread chan<- *Thread) {
	if lc := eal.CurrentLCore(); lc != eal.Initial {
		log.Panicf("lc=%v is not initail=%v", lc, eal.Initial)
	}

	var e error
	mainThread, e = NewThread()
	if e != nil {
		log.Fatalf("SPDK thread error %s", e)
		return
	}
	mainThread.SetLCore(eal.Initial)
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
