// Package strategycode manages forwarding strategy BPF programs.
package strategycode

/*
#include "../../csrc/strategycode/strategy-code.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Strategy is a reference of a forwarding strategy BPF program.
type Strategy C.StrategyCode

// Ptr returns *C.Strategy pointer.
func (sc *Strategy) Ptr() unsafe.Pointer {
	return unsafe.Pointer(sc)
}

func (sc *Strategy) ptr() *C.StrategyCode {
	return (*C.StrategyCode)(sc)
}

// ID returns numeric ID.
func (sc *Strategy) ID() int {
	return int(sc.ptr().id)
}

// Name returns short name.
func (sc *Strategy) Name() string {
	return C.GoString(sc.ptr().name)
}

// CountRefs returns number of references.
// Each FIB entry using the strategy has a reference.
// There's also a reference from table.go.
func (sc *Strategy) CountRefs() int {
	return int(sc.ptr().nRefs)
}

// Close reduces the number of references by one.
// The strategy will be unloaded when its reference count reaches zero.
func (sc *Strategy) Close() error {
	tableLock.Lock()
	defer tableLock.Unlock()
	delete(table, sc.ID())
	C.StrategyCode_Unref(sc.ptr())
	return nil
}

func (sc *Strategy) String() string {
	if sc == nil {
		return "0@nil"
	}
	return fmt.Sprintf("%d@%p", sc.ID(), sc.ptr())
}

// FromPtr converts *C.Strategy to *Strategy.
func FromPtr(ptr unsafe.Pointer) *Strategy {
	return (*Strategy)(ptr)
}
