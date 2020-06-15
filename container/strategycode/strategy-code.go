package strategycode

/*
#include "../../csrc/strategycode/strategy-code.h"
*/
import "C"
import (
	"fmt"
	"io"
	"unsafe"
)

// BPF program of a forwarding strategy.
type StrategyCode interface {
	fmt.Stringer
	io.Closer
	GetPtr() unsafe.Pointer
	GetId() int
	GetName() string
	CountRefs() int
}

type scImpl struct {
	c *C.StrategyCode
}

// Retrieve *C.StrategyCode pointer.
func (sc *scImpl) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(sc.c)
}

// Get numeric ID.
func (sc *scImpl) GetId() int {
	return int(sc.c.id)
}

// Get short name.
func (sc *scImpl) GetName() string {
	return C.GoString(sc.c.name)
}

// Get number of references, including a reference from table.go.
func (sc *scImpl) CountRefs() int {
	return int(sc.c.nRefs)
}

// Unreference. Strategy will be unloaded when no FIB entry is using it.
func (sc *scImpl) Close() error {
	tableLock.Lock()
	defer tableLock.Unlock()
	delete(table, sc.GetId())
	C.StrategyCode_Unref(sc.c)
	return nil
}

func (sc *scImpl) String() string {
	if sc == nil {
		return "0@nil"
	}
	return fmt.Sprintf("%d@%p", sc.GetId(), sc.c)
}
