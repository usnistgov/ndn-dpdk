// Package strategycode manages forwarding strategy BPF programs.
package strategycode

import (
	"fmt"
	"sync"
)

// Table of Strategy instances.
var (
	lastID    int
	table     = map[int]*Strategy{}
	tableLock sync.Mutex
)

// Get retrieves strategy by numeric ID.
func Get(id int) *Strategy {
	tableLock.Lock()
	defer tableLock.Unlock()
	if sc := table[id]; sc != nil {
		// Directly 'return table[id]' would return interface with nil underlying
		// value. This conditional prevents that.
		return sc
	}
	return nil
}

// Find retrieves strategy by name.
func Find(name string) *Strategy {
	tableLock.Lock()
	defer tableLock.Unlock()
	for _, sc := range table {
		if sc.Name() == name {
			return sc
		}
	}
	return nil
}

// List returns a list of loaded strategies.
func List() []*Strategy {
	tableLock.Lock()
	defer tableLock.Unlock()
	list := make([]*Strategy, 0, len(table))
	for _, sc := range table {
		list = append(list, sc)
	}
	return list
}

// DestroyAll immediately unloads all strategies.
// Panics if some strategies are still used in FIB entry.
func DestroyAll() {
	for _, sc := range List() {
		if nRefs := sc.CountRefs(); nRefs > 1 {
			panic(fmt.Errorf("%s has %d refs", sc, nRefs))
		}
		sc.Unref()
	}
}
