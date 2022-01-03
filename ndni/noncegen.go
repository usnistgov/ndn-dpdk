package ndni

/*
#include "../csrc/ndni/interest.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"
)

// InitNonceGen initializes *C.NonceGen.
func InitNonceGen(g unsafe.Pointer) {
	c := (*C.NonceGen)(g)
	C.pcg32_srandom_r(&c.rng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
}
