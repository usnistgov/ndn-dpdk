package strategycode

/*
#include "strategy-code.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// BPF program of a forwarding strategy.
type StrategyCode struct {
	c *C.StrategyCode
}

func (sc StrategyCode) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(sc.c)
}

func FromPtr(ptr unsafe.Pointer) StrategyCode {
	return StrategyCode{(*C.StrategyCode)(ptr)}
}

func (sc StrategyCode) GetId() int {
	return int(sc.c.id)
}

func (sc StrategyCode) GetName() string {
	return C.GoString(sc.c.name)
}

func (sc StrategyCode) CountRefs() int {
	return int(sc.c.nRefs)
}

func (sc StrategyCode) Ref() {
	C.StrategyCode_Ref(sc.c)
}

func (sc StrategyCode) Unref() {
	C.StrategyCode_Unref(sc.c)
}

func (sc StrategyCode) Close() error {
	if sc.CountRefs() > 0 {
		return errors.New("StrategyCode has references")
	}

	tableLock.Lock()
	defer tableLock.Unlock()
	C.rte_bpf_destroy(sc.c.bpf)
	delete(table, sc.GetId())
	C.free(unsafe.Pointer(sc.c.name))
	dpdk.Free(sc.c)
	return nil
}

func (sc StrategyCode) String() string {
	if sc.c == nil {
		return "0@nil"
	}
	return fmt.Sprintf("%d@%p", sc.GetId(), sc.c)
}
