// Package eal contains bindings of DPDK Environment Abstraction Layer.
package eal

import (
	"math/rand"
	"sort"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
)

var logger = logging.New("eal")

// Version is DPDK version.
// This is populated by package ealinit.
var Version string

// EAL variables, available after ealinit.Init().
var (
	// MainLCore is the main lcore.
	MainLCore LCore
	// Workers are worker lcores.
	Workers LCores
	// Sockets are NUMA sockets of worker lcores.
	Sockets []NumaSocket

	// MainThread is a PollThread running on the initial lcore.
	MainThread PollThread
	// MainReadSide is an RCU read-side object of the MainThread.
	MainReadSide *urcu.ReadSide
)

// UpdateLCoreSockets saves LCores and Sockets information.
// If this is used for mocking in unit tests, an undo function is provided to revert the changes.
func UpdateLCoreSockets(lcoreSockets map[int]int, mainLCoreID int) (undo func()) {
	oldMainLCore, oldWorkers, oldSockets, oldLCoreToSocket := MainLCore, Workers, Sockets, lcoreToSocket
	undo = func() {
		MainLCore, Workers, Sockets, lcoreToSocket = oldMainLCore, oldWorkers, oldSockets, oldLCoreToSocket
	}

	MainLCore, Workers, Sockets = LCoreFromID(mainLCoreID), nil, nil

	socketIDs := map[int]bool{}
	for lcID, socketID := range lcoreSockets {
		lcoreToSocket[lcID] = socketID
		socketIDs[socketID] = true
		if lcID != mainLCoreID {
			Workers = append(Workers, LCoreFromID(lcID))
		}
	}
	sort.Slice(Workers, func(i, j int) bool { return Workers[i].v < Workers[j].v })

	for socketID := range socketIDs {
		Sockets = append(Sockets, NumaSocketFromID(socketID))
	}

	sort.Slice(Sockets, func(i, j int) bool { return Sockets[i].v < Sockets[j].v })

	return
}

// RandomSocket returns a random NumaSocket that has at least one worker lcore.
func RandomSocket() (socket NumaSocket) {
	if n := len(Sockets); n > 0 {
		return Sockets[rand.Intn(n)]
	}
	return NumaSocket{}
}

// PollThread represents a thread that can accept and execute posted functions.
type PollThread interface {
	Post(fn cptr.Function)
}

// PostMain asynchronously posts a function to be executed on the main thread.
func PostMain(fn cptr.Function) {
	MainThread.Post(fn)
}

// CallMain executes a function on the main thread and waits for its completion.
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func CallMain(f interface{}) interface{} {
	if CurrentLCore() == MainLCore {
		return cptr.Call(func(fn cptr.Function) { cptr.Func0.Invoke(fn) }, f)
	}
	return cptr.Call(PostMain, f)
}
