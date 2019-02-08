package spdk

/*
#include <spdk/env_dpdk.h>
#include <spdk/log.h>
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/core/dlopen"
	"ndn-dpdk/core/logger"
	"ndn-dpdk/dpdk"
)

// SPDK thread for most operations invoked from Go API.
var MainThread *Thread

// Initialize SPDK environment and create a main thread.
func Init(eal *dpdk.Eal, mainThreadLcore dpdk.LCore) (e error) {
	if MainThread != nil && MainThread.IsRunning() { // already initialized
		return nil
	}

	if e = dlopen.LoadDynLibs("/usr/local/lib/libspdk.so"); e != nil {
		return e
	}

	if res := int(C.spdk_env_dpdk_post_init()); res != 0 {
		return dpdk.Errno(-res)
	}

	initLogging()

	if MainThread, e = NewThread("SPDK-main"); e != nil {
		return e
	}
	MainThread.SetLCore(mainThreadLcore)
	if e = MainThread.Launch(); e != nil {
		return e
	}

	if e = initRpc(); e != nil {
		return e
	}

	return nil
}

func MustInit(eal *dpdk.Eal, mainThreadLcore dpdk.LCore) {
	if e := Init(eal, mainThreadLcore); e != nil {
		panic(fmt.Sprintf("spdk.Init error %v", e))
	}
}

func initLogging() {
	lvl := logger.GetLevel("SPDK")
	lvlC := C.enum_spdk_log_level(C.SPDK_LOG_INFO)
	switch lvl {
	case 'V':
		lvlC = C.SPDK_LOG_DEBUG
	case 'D':
		lvlC = C.SPDK_LOG_INFO
	case 'I':
		lvlC = C.SPDK_LOG_NOTICE
	case 'W':
		lvlC = C.SPDK_LOG_WARN
	case 'E', 'F':
		lvlC = C.SPDK_LOG_ERROR
	case 'N':
		lvlC = C.SPDK_LOG_DISABLED
	}
	C.spdk_log_set_print_level(lvlC)
}
