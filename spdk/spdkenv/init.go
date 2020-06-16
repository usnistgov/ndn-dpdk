package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/env_dpdk.h>
*/
import "C"
import (
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/dlopen"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var (
	// MainThread is an SPDK thread for most operations invoked from Go API.
	MainThread *Thread

	spdkInitOnce sync.Once
)

// Init initializes the SPDK environment and creates a main thread.
// Errors are fatal.
func Init(mainThreadLcore eal.LCore) {
	spdkInitOnce.Do(func() {
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

		if MainThread, e = NewThread("SPDK-main"); e != nil {
			log.Fatalf("SPDK thread error %s", e)
			return
		}
		MainThread.SetLCore(mainThreadLcore)
		if e = MainThread.Launch(); e != nil {
			log.Fatalf("SPDK launch error %s", e)
			return
		}

		if e = initRPC(); e != nil {
			log.Fatalf("SPDK RPC init error %s", e)
			return
		}
	})
}
