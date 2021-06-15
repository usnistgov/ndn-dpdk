package ealthreadtest

/*
#include "../../../csrc/dpdk/thread.h"

typedef struct TestThread {
	ThreadLoadStat loadStat;
	int n;
	ThreadStopFlag stop;
} TestThread;

int
TestThread_Run(TestThread* th) {
	th->n = 0;
	while (ThreadStopFlag_ShouldContinue(&th->stop)) {
		++th->n;
		ThreadLoadStat_Report(&th->loadStat, th->n % 5);
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
	ealthread.Thread
	c *C.TestThread
}

var (
	_ ealthread.ThreadWithRole     = (*testThread)(nil)
	_ ealthread.ThreadWithLoadStat = (*testThread)(nil)
)

func newTestThread() *testThread {
	var th testThread
	th.c = (*C.TestThread)(eal.Zmalloc("TestThread", C.sizeof_TestThread, eal.NumaSocket{}))
	th.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TestThread_Run), unsafe.Pointer(th.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&th.c.stop)),
	)
	return &th
}

func (th *testThread) ThreadRole() string {
	return "TEST"
}

func (th *testThread) ThreadLoadStat() ealthread.LoadStat {
	return ealthread.LoadStatFromPtr(unsafe.Pointer(&th.c.loadStat))
}

func (th *testThread) GetN() int {
	return int(th.c.n)
}

func (th *testThread) Close() error {
	eal.Free(th.c)
	return nil
}
