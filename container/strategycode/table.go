// Package strategycode manages forwarding strategy BPF programs.
package strategycode

import (
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"go.uber.org/zap"
)

var logger = logging.New("strategycode")

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
	return table[id]
}

// Find retrieves strategy by name.
func Find(name string) *Strategy {
	tableLock.Lock()
	defer tableLock.Unlock()
	for _, sc := range table {
		if sc.name == name {
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
			logger.Panic("DestroyAll strategy has non-zero refs",
				zap.Int("id", sc.ID()),
				zap.Int("refs", nRefs),
			)
		}
		sc.Unref()
	}
}
