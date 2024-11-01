package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/pcg32"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type worker struct {
	ealthread.ThreadWithCtrl
	c      *C.FetchThread
	index  C.int8_t
	nTasks int
}

var (
	_ ealthread.ThreadWithRole     = (*worker)(nil)
	_ ealthread.ThreadWithLoadStat = (*worker)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return tgdef.RoleConsumer
}

func (w *worker) Face() iface.Face {
	return iface.Get(iface.ID(w.c.face))
}

// NumaSocket implements eal.WithNumaSocket interface.
func (w *worker) NumaSocket() eal.NumaSocket {
	return w.Face().NumaSocket()
}

// AddTask adds a task.
// ts must refer to an inactive task slot.
func (w *worker) AddTask(rs *urcu.ReadSide, ts *taskSlot) {
	rs.Lock()
	defer rs.Unlock()

	if ts.worker != -1 {
		logger.Panic("worker.AddTask called with active task")
	}
	ts.worker = w.index

	w.nTasks++
	C.cds_hlist_add_head_rcu(&ts.fthNode, &w.c.tasksHead)
}

// RemoveTask removes a task.
// ts must refer to a task slot added to this worker.
func (w *worker) RemoveTask(rs *urcu.ReadSide, ts *taskSlot) {
	rs.Lock()
	defer rs.Unlock()

	if ts.worker != w.index {
		logger.Panic("worker.RemoveTask called with task not active on this worker")
	}
	ts.worker = -1

	w.nTasks--
	C.cds_hlist_del_rcu(&ts.fthNode)
	urcu.Synchronize()
}

// ClearTasks clears task list.
// This is non-thread-safe.
func (w *worker) ClearTasks() {
	w.c.tasksHead.next = nil
	w.nTasks = 0
}

func newWorker(face iface.Face, index int) (w *worker) {
	socket := face.NumaSocket()
	w = &worker{
		c:     eal.Zmalloc[C.FetchThread]("FetchThread", C.sizeof_FetchThread, socket),
		index: C.int8_t(index),
	}
	*w.c = C.FetchThread{
		interestMp: (*C.struct_rte_mempool)(ndni.InterestMempool.Get(socket).Ptr()),
		face:       C.FaceID(face.ID()),
	}
	w.c.uringCapacity, w.c.uringWaitLbound = 4096, 2048
	pcg32.Init(unsafe.Pointer(&w.c.nonceRng))
	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.FetchThread_Run, w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	return w
}
