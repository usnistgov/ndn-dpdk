package urcu

/*
#include "../../csrc/core/urcu.h"
*/
import "C"

// Synchronize invokes synchronize_rcu.
func Synchronize() {
	C.synchronize_rcu()
}

// Barrier declares an RCU barrier.
func Barrier() {
	C.rcu_barrier()
}
