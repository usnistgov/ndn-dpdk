// Package spdkenv contains bindings of SPDK environment and threads.
package spdkenv

/*
#include "../../csrc/core/logger.h"
#include <spdk/env_dpdk.h>
#include <spdk/init.h>
#include <spdk/log.h>
#include <spdk/version.h>

static void c_SpdkLoggerReady()
{
	SPDK_NOTICELOG("SPDK logger ready\n");
}
*/
import "C"
import (
	"fmt"
	"strings"

	"github.com/usnistgov/ndn-dpdk/core/dlopen"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var logger = logging.New("spdkenv")

// Version is SPDK version.
var Version = strings.TrimPrefix(C.SPDK_VERSION_STRING, "SPDK v")

var mainThread *Thread

// InitEnv initializes the SPDK environment.
func InitEnv() error {
	// As of SPDK 21.10, libspdk_scheduler_dpdk_governor.so depends on rte_power_freq_max symbol
	// exported by librte_power.so but does not link with that library.
	dlopen.Load("/usr/local/lib/librte_power.so")

	e := dlopen.LoadGroup("/usr/local/lib/libspdk.so")
	if e != nil {
		return fmt.Errorf("dlopen(libspdk.so) error: %w", e)
	}

	C.spdk_log_open((*C.logfunc)(C.Logger_Spdk))

	if res := int(C.spdk_env_dpdk_post_init(C.bool(false))); res != 0 {
		return fmt.Errorf("spdk_env_dpdk_post_init error: %w", eal.MakeErrno(res))
	}

	C.c_SpdkLoggerReady()
	return nil
}

// InitMainThread creates a main thread, and launches on the current goroutine.
// This must be invoked on the MainLCore.
// This function never returns; either the main thread (*Thread) or an error is sent to `ret`.
func InitMainThread(ret chan<- any) {
	if lc := eal.CurrentLCore(); lc != eal.MainLCore {
		logger.Panic("lcore is not main", lc.ZapField("lc"), eal.MainLCore.ZapField("main"))
	}

	var e error
	mainThread, e = NewThread()
	if e != nil {
		ret <- fmt.Errorf("SPDK thread error: %w", e)
		return
	}
	mainThread.SetLCore(eal.MainLCore)
	ret <- mainThread
	mainThread.main()
}

// InitFinal finishes initializing SPDK.
func InitFinal() (e error) {
	e, _ = eal.CallMain(initRPC).(error)
	if e != nil {
		return fmt.Errorf("SPDK RPC init error: %w", e)
	}
	return nil
}
