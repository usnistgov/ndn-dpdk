package urcu

/*
#include "../../csrc/core/urcu.h"
*/
import "C"

func Synchronize() {
	C.synchronize_rcu()
}

func Barrier() {
	C.rcu_barrier()
}
