// Package pcg32 interacts with PCG random number generators.
package pcg32

/*
#include "../../csrc/vendor/pcg_basic.h"
*/
import "C"
import (
	"math/rand/v2"
	"unsafe"
)

// Init initializes *C.pcg32_random_t.
func Init(ptr unsafe.Pointer) {
	C.pcg32_srandom_r((*C.pcg32_random_t)(ptr), C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
}
