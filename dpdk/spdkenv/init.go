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
	"errors"
	"fmt"
	"path/filepath"
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
	if e := loadLibspdk(); e != nil {
		return e
	}

	C.spdk_log_open((*C.spdk_log_cb)(C.Logger_Spdk))

	if res := int(C.spdk_env_dpdk_post_init(C.bool(false))); res != 0 {
		return fmt.Errorf("spdk_env_dpdk_post_init error: %w", eal.MakeErrno(res))
	}

	C.c_SpdkLoggerReady()
	return nil
}

func loadLibspdk() error {
	errs := []error{}

	// As of SPDK 25.01-rc1, libspdk_scheduler_dpdk_governor.so depends on rte_power_freq_max symbol
	// exported by librte_power.so but does not link with that library.
	if _, e := dlopen.Load("librte_power.so"); e != nil {
		errs = append(errs, fmt.Errorf("dlopen(librte_power.so): %w", e))
	}

	for _, libdir := range []string{"/usr/local/lib", "/usr/lib"} {
		filename := filepath.Join(libdir, "libspdk.so")
		if e := dlopen.LoadGroup(filename); e != nil {
			errs = append(errs, fmt.Errorf("dlopen(%s): %w", filename, e))
		} else {
			return nil
		}
	}

	return errors.Join(errs...)
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
