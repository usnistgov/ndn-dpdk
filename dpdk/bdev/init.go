package bdev

/*
#include "../../csrc/dpdk/bdev.h"
#include <spdk/accel.h>

extern void go_bdevInitialized(void* ctx, int rc);
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

// RequiredBlockSize is the expected block size.
const RequiredBlockSize = C.BdevBlockSize

var initBdevLibOnce sync.Once

// Initialize SPDK block device library.
func initBdevLib() {
	initBdevLibOnce.Do(func() {
		logger.Info("initializing block device library and accel framework")
		eal.CallMain(func() {
			C.spdk_accel_initialize()
			C.spdk_bdev_initialize(C.spdk_bdev_init_cb(C.go_bdevInitialized), nil)
		})
	})
}

//export go_bdevInitialized
func go_bdevInitialized(_ unsafe.Pointer, rc C.int) {
	if rc != 0 {
		logger.Panic("spdk_bdev_initialize error", zap.Error(eal.MakeErrno(rc)))
	}
	C.BdevFiller_ = eal.ZmallocAligned[C.uint8_t]("BdevFiller", C.UINT16_MAX+1, 4096/C.RTE_CACHE_LINE_SIZE, eal.NumaSocket{})
}
