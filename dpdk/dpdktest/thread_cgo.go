package dpdktest

/*
#include "../thread.h"

typedef struct TestThread {
	int n;
	ThreadStopFlag stop;
} TestThread;

void
TestThread_Run(TestThread* thread) {
	thread->n = 0;
	while (ThreadStopFlag_ShouldContinue(&thread->stop)) {
		++thread->n;
	}
}
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type testThread struct {
	dpdk.ThreadBase
	c *C.TestThread
}

func newTestThread() (th *testThread) {
	th = new(testThread)
	th.ResetThreadBase()
	th.c = (*C.TestThread)(dpdk.Zmalloc("TestThread", C.sizeof_TestThread, dpdk.NUMA_SOCKET_ANY))
	dpdk.InitStopFlag(unsafe.Pointer(&th.c.stop))
	return th
}

func (th *testThread) GetN() int {
	return int(th.c.n)
}

func (th *testThread) Launch() error {
	return th.LaunchImpl(func() int {
		C.TestThread_Run(th.c)
		return 0
	})
}

func (th *testThread) Stop() error {
	return th.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&th.c.stop)))
}

func (th *testThread) Close() error {
	dpdk.Free(th.c)
	return nil
}
