package bdev

/*
#include "../../csrc/spdk/bdev.h"
#include <spdk/accel_engine.h>

extern void go_bdevInitialized(void* ctx, int rc);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/spdk/spdkenv"
)

var initBdevLibOnce sync.Once

// Initialize SPDK block device library.
func initBdevLib() {
	initBdevLibOnce.Do(func() {
		spdkenv.MainThread.Call(func() { C.spdk_bdev_initialize(C.spdk_bdev_init_cb(C.go_bdevInitialized), nil) })
	})
}

//export go_bdevInitialized
func go_bdevInitialized(ctx unsafe.Pointer, rc C.int) {
	if rc != 0 {
		panic(fmt.Sprintf("spdk_bdev_initialize error %v", eal.Errno(rc)))
	}
	C.SpdkBdev_InitFiller()
}

var initAccelEngineOnce sync.Once

func initAccelEngine() {
	initAccelEngineOnce.Do(func() {
		spdkenv.MainThread.Call(func() { C.spdk_accel_engine_initialize() })
	})
}
