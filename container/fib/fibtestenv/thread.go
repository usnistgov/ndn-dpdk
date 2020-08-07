package fibtestenv

import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// LookupThread implements fib.LookupThread.
type LookupThread struct {
	Socket  eal.NumaSocket
	Replica unsafe.Pointer
	Index   int
}

// NumaSocket returns th.Socket.
func (th *LookupThread) NumaSocket() eal.NumaSocket {
	return th.Socket
}

// SetFib saves arguments to instance.
func (th *LookupThread) SetFib(replica unsafe.Pointer, i int) {
	th.Replica = replica
	th.Index = i
}
