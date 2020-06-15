package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"strconv"
	"sync"

	"github.com/jaypipes/ghw"
)

// NumaSocket represents a NUMA socket.
// Zero value is SOCKET_ID_ANY.
type NumaSocket struct {
	v int // socket ID + 1
}

var (
	numaSocketListInit sync.Once
	numaSocketList     []NumaSocket
)

// ListNumaSockets returns a list of NumaSockets.
// Note that not every NumaSocket is used in DPDK.
func ListNumaSockets() []NumaSocket {
	numaSocketListInit.Do(func() {
		topology, e := ghw.Topology()
		if e != nil {
			return
		}
		for _, node := range topology.Nodes {
			numaSocketList = append(numaSocketList, NumaSocketFromID(node.ID))
		}
	})
	return numaSocketList
}

// NumaSocketFromID converts socket ID to NumaSocket.
func NumaSocketFromID(id int) (socket NumaSocket) {
	if id < 0 || id > C.RTE_MAX_NUMA_NODES {
		return socket
	}
	socket.v = id + 1
	return socket
}

// ID returns NUMA socket ID.
func (socket NumaSocket) ID() int {
	return socket.v - 1
}

// IsAny returns true if this represents SOCKET_ID_ANY.
func (socket NumaSocket) IsAny() bool {
	return socket.v == 0
}

func (socket NumaSocket) String() string {
	if socket.IsAny() {
		return "any"
	}
	return strconv.Itoa(socket.ID())
}
