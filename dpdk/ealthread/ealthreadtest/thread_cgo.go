package ealthreadtest

/*
#include "../../../csrc/dpdk/thread.h"

typedef struct TestThread {
	ThreadCtrl ctrl;
	int n;
} TestThread;

int
TestThread_Run(TestThread* th) {
	th->n = 0;
	int x = 0;
	while (ThreadCtrl_Continue(th->ctrl, x)) {
		++th->n;
		x = th->n % 5;
	}
	return 0;
}
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

type testThread struct {
	ealthread.ThreadWithCtrl
	c *C.TestThread
}

var _ interface {
	ealthread.ThreadWithRole
	ealthread.ThreadWithLoadStat
	ealthread.ThreadWithCtrl
} = &testThread{}

func newTestThread() *testThread {
	var th testThread
	th.c = eal.Zmalloc[C.TestThread]("TestThread", C.sizeof_TestThread, eal.NumaSocket{})
	th.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.TestThread_Run, th.c),
		unsafe.Pointer(&th.c.ctrl),
	)
	return &th
}

func (th *testThread) ThreadRole() string {
	return "TEST"
}

func (th *testThread) GetN() int {
	return int(th.c.n)
}

func (th *testThread) Close() error {
	eal.Free(th.c)
	return nil
}
