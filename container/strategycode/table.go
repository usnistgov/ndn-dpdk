package strategycode

/*
#include "strategy-code.h"
*/
import "C"
import (
	"sync"
)

// Table of StrategyCode instances.
var (
	lastId    int
	table     map[int]StrategyCode = make(map[int]StrategyCode)
	tableLock sync.Mutex
)

func Get(id int) (sc StrategyCode, ok bool) {
	tableLock.Lock()
	defer tableLock.Unlock()
	sc, ok = table[id]
	return
}

func Find(name string) (sc StrategyCode, ok bool) {
	tableLock.Lock()
	defer tableLock.Unlock()
	for _, sc = range table {
		if sc.GetName() == name {
			return sc, true
		}
	}
	return StrategyCode{}, false
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

func CloseAll() {
	for _, sc := range List() {
		sc.Close()
	}
}
