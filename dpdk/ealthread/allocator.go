package ealthread

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var allocated [eal.MaxLCoreID]string

// AllocReq represents a request to the allocator.
type AllocReq struct {
	Role   string         // role name, must not be empty
	Socket eal.NumaSocket // preferred NUMA socket
}

func lcAllocatedTo(role string) eal.LCorePredicate {
	return func(lc eal.LCore) bool {
		return allocated[lc.ID()] == role
	}
}

func lcUnallocated() eal.LCorePredicate {
	return lcAllocatedTo("")
}

// AllocConfig allocates lcores according to configuration.
func AllocConfig(c Config) (m map[string]eal.LCores, e error) {
	m, e = c.assignWorkers(lcUnallocated())
	if e == nil {
		for role, lcores := range m {
			for _, lc := range lcores {
				allocated[lc.ID()] = role
			}
			logger.Info("lcores configured", zap.String("role", role), zap.Array("lc", lcores))
		}
	}
	return m, e
}

// AllocRequest allocates lcores to requests.
// Any request with Role=="" is skipped.
func AllocRequest(requests ...AllocReq) (list []eal.LCore, e error) {
	reqBySocket, reqAny, nReq := map[eal.NumaSocket][]int{}, []int{}, 0
	for i, req := range requests {
		if req.Role == "" {
			continue
		}
		nReq++
		if req.Socket.IsAny() {
			reqAny = append(reqAny, i)
		} else {
			reqBySocket[req.Socket] = append(reqBySocket[req.Socket], i)
		}
	}

	workers := eal.Workers.Filter(lcUnallocated())
	if nAvail := len(workers); nReq > nAvail {
		return nil, fmt.Errorf("need %d lcores but only %d available", nReq, nAvail)
	}
	list = make([]eal.LCore, len(requests))

	workersBySocket := workers.ByNumaSocket()
	take := func(socket eal.NumaSocket) eal.LCore {
		lc := workersBySocket[socket][0]
		workersBySocket[socket] = workersBySocket[socket][1:]
		return lc
	}
	for socket, reqIndexes := range reqBySocket {
		for _, i := range reqIndexes {
			if len(workersBySocket[socket]) > 0 {
				list[i] = take(socket)
			} else {
				reqAny = append(reqAny, i)
			}
		}
	}

	sockets := append([]eal.NumaSocket{}, eal.Sockets...)
	for _, i := range reqAny {
		slices.SortFunc(sockets, func(a, b eal.NumaSocket) bool { return len(workersBySocket[a]) > len(workersBySocket[b]) })
		list[i] = take(sockets[0]) // pick from least occupied NUMA socket
	}

	for i, req := range requests {
		if req.Role == "" {
			continue
		}
		lc := list[i]
		allocated[lc.ID()] = req.Role
		logger.Info("lcore requested", zap.String("role", req.Role), req.Socket.ZapField("req-socket"), lc.ZapField("lc"))
	}

	return list, nil
}

// AllocFree deallocates an lcore.
func AllocFree(lcores ...eal.LCore) {
	allocFree(lcores, false)
}

// AllocClear deletes all allocations.
//
// When this is used together with eal.UpdateLCoreSockets mock in a test case, the cleanup statement should be ordered as:
//  defer eal.UpdateLCoreSockets(...)()
//  defer ealthread.AllocClear()
// This ensures AllocClear() clears allocations for mocked workers instead of real workers.
func AllocClear() {
	allocFree(eal.Workers, true)
}

func allocFree(lcores []eal.LCore, maybeFree bool) {
	var freed eal.LCores
	for _, lc := range lcores {
		role := allocated[lc.ID()]
		if role != "" {
			freed = append(freed, lc)
			allocated[lc.ID()] = ""
		} else if !maybeFree {
			logger.Panic("lcore double free", lc.ZapField("lc"))
		}
	}
	if len(freed) > 0 {
		logger.Info("lcores freed", zap.Array("lc", freed))
	}
}
