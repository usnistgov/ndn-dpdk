// Package eal contains bindings of DPDK Environment Abstraction Layer.
package eal

import (
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/zyedidia/generic/mapset"
	"golang.org/x/exp/slices"
)

var logger = logging.New("eal")

// Version is DPDK version.
// This is assigned during package ealinit initialization.
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
// Returns an undo function that reverts the changes, useful for mocking in unit tests.
func UpdateLCoreSockets(lcoreSockets map[int]int, mainLCoreID int) (undo func()) {
	oldMainLCore, oldWorkers, oldSockets, oldLCoreToSocket := MainLCore, Workers, Sockets, lcoreToSocket
	undo = func() {
		MainLCore, Workers, Sockets, lcoreToSocket = oldMainLCore, oldWorkers, oldSockets, oldLCoreToSocket
	}

	MainLCore, Workers, Sockets = LCoreFromID(mainLCoreID), nil, nil

	socketIDs := mapset.New[int]()
	for lcID, socketID := range lcoreSockets {
		lcoreToSocket[lcID] = socketID
		socketIDs.Put(socketID)
		if lcID != mainLCoreID {
			Workers = append(Workers, LCoreFromID(lcID))
		}
	}
	slices.SortFunc(Workers, func(a, b LCore) bool { return a.v < b.v })

	socketIDs.Each(func(socketID int) {
		Sockets = append(Sockets, NumaSocketFromID(socketID))
	})
	slices.SortFunc(Sockets, func(a, b NumaSocket) bool { return a.v < b.v })

	return
}

// RandomSocket returns a random NumaSocket that has at least one worker lcore.
func RandomSocket() (socket NumaSocket) {
	if n := len(Sockets); n > 0 {
		return Sockets[rand.Intn(n)]
	}
	return NumaSocket{}
}

// PollThread represents a thread that can accept and run posted functions.
type PollThread interface {
	Post(fn cptr.Function)
}

// PostMain asynchronously posts a function to be run on the main thread.
func PostMain(fn cptr.Function) {
	MainThread.Post(fn)
}

// CallMain runs a function on the main thread and waits for its completion.
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func CallMain(f any) any {
	if CurrentLCore() == MainLCore {
		return cptr.Call(func(fn cptr.Function) { cptr.Func0.Invoke(fn) }, f)
	}
	return cptr.Call(PostMain, f)
}
