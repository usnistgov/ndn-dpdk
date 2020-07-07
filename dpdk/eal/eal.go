package eal

import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// LCore and NUMA sockets, available after ealinit.Init().
var (
	Initial LCore
	Workers []LCore
	Sockets []NumaSocket

	MainThread PollThread
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
	return cptr.Call(PostMain, f)
}
