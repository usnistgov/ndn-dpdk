package urcu

/*
#include "urcu.h"
*/
import "C"

func Synchronize() {
	C.synchronize_rcu()
}

func Barrier() {
	C.rcu_barrier()
}
