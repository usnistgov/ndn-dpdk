package fibtestenv

import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// LookupThread is a trivial implementation of fib.LookupThread interface.
type LookupThread struct {
	Socket   eal.NumaSocket
	Replica  unsafe.Pointer
	DynIndex int
}

// NumaSocket returns th.Socket.
func (th *LookupThread) NumaSocket() eal.NumaSocket {
	return th.Socket
}

// GetFibSgGlobal returns nil.
// This may not work with strategy with SgInit function.
func (th *LookupThread) GetFibSgGlobal() unsafe.Pointer {
	return nil
}

// GetFib returns saved arguments.
func (th *LookupThread) GetFib() (replica unsafe.Pointer, dynIndex int) {
	return th.Replica, th.DynIndex
}

// SetFib saves arguments to instance.
func (th *LookupThread) SetFib(replica unsafe.Pointer, dynIndex int) {
	th.Replica = replica
	th.DynIndex = dynIndex
}
