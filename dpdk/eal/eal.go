// Package eal contains bindings of DPDK Environment Abstraction Layer.
package eal

import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
)

// EAL variables, available after ealinit.Init().
var (
	// Initial is the initial lcore.
	Initial LCore
	// Workers are worker lcores.
	Workers []LCore
	// Sockets are NUMA sockets of worker lcores.
	Sockets []NumaSocket

	// MainThread is a PollThread running on the initial lcore.
	MainThread PollThread
	// MainReadSide is an RCU read-side object of the MainThread.
	MainReadSide *urcu.ReadSide
)

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
	if CurrentLCore() == Initial {
		return cptr.Call(func(fn cptr.Function) { cptr.Func0.Invoke(fn) }, f)
	}
	return cptr.Call(PostMain, f)
}
