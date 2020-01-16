package strategycode

/*
#include "strategy-code.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// Table of StrategyCode instances.
var (
	lastId    int
	table     = make(map[int]*scImpl)
	tableLock sync.Mutex
)

// Retrieve by numeric ID.
func Get(id int) StrategyCode {
	tableLock.Lock()
	defer tableLock.Unlock()
	if sc := table[id]; sc != nil {
		// Directly 'return table[id]' would return interface with nil underlying
		// value. This conditional prevents that.
		return sc
	}
	return nil
}

// Retrieve by name.
func Find(name string) StrategyCode {
	tableLock.Lock()
	defer tableLock.Unlock()
	for _, sc := range table {
		if sc.GetName() == name {
			return sc
		}
	}
	return nil
}

// Retrieve by pointer.
func FromPtr(ptr unsafe.Pointer) StrategyCode {
	tableLock.Lock()
	defer tableLock.Unlock()
	for _, sc := range table {
		if sc.c == (*C.StrategyCode)(ptr) {
			return sc
		}
	}
	return nil
}

func List() []StrategyCode {
	tableLock.Lock()
	defer tableLock.Unlock()
	list := make([]StrategyCode, 0, len(table))
	for _, sc := range table {
		list = append(list, sc)
	}
	return list
}

// Immediately unload all strategies.
// Panics if some strategies are still used in FIB entry.
func DestroyAll() {
	for _, sc := range List() {
		if nRefs := sc.CountRefs(); nRefs > 1 {
			panic(fmt.Errorf("%s has %d refs", sc, nRefs))
		}
		sc.Close()
	}
}
