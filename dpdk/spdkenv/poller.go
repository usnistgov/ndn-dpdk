package spdkenv

/*
#include "../../csrc/dpdk/spdk-thread.h"
*/
import "C"
import (
	"time"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// Poller periodically executes a function on an SPDK thread.
type Poller struct {
	c      *C.struct_spdk_poller
	th     *Thread
	revoke func()
}

// NewPoller creates a Poller.
func NewPoller(th *Thread, fn cptr.Function, d time.Duration) *Poller {
	f, arg, revoke := cptr.Func0.CallbackReuse(fn)
	poller := &Poller{
		th:     th,
		revoke: revoke,
	}
	cptr.Call(th.Post, func() {
		poller.c = C.spdk_poller_register(C.spdk_poller_fn(f), arg, C.uint64_t(d/time.Microsecond))
	})
	return poller
}

// Stop cancels periodical execution.
func (poller *Poller) Stop() {
	cptr.Call(poller.th.Post, func() { C.spdk_poller_unregister(&poller.c) })
	poller.revoke()
}
