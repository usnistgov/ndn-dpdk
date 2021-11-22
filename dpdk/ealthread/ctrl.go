package ealthread

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_DPDK_THREAD_ENUM_H -out=../../csrc/dpdk/thread-enum.h .

/*
#include "../../csrc/dpdk/thread.h"

#ifdef NDNDPDK_THREADSLEEP
#define ENABLE_THREADSLEEP 1
#else
#define ENABLE_THREADSLEEP 0
#endif
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

const sleepEnabled = C.ENABLE_THREADSLEEP > 0

// ThreadSleep constants.
// These are effective only if NDNDPDK_MK_THREADSLEEP=1 is set during compilation.
// If a C thread had an empty poll, i.e. processed zero packets or work items during an iteration,
// it sleeps for a short duration to reduce CPU utilization.
//
// Initially, the sleep duration is SleepMin.
// Once every SleepAdjustEvery consecutive empty polls, the sleep duration is adjusted as:
//   d = MIN(SleepMax, d * SleepMultiply / SleepDivide + SleepAdd)
// After a valid poll, i.e. processed non-zero packets, the sleep duration is reset to SleepMin.
//
// All durations are in nanoseconds unit.
const (
	SleepMin         = 1
	SleepMax         = 100000
	SleepAdjustEvery = 1 << 10
	SleepMultiply    = 11
	SleepDivide      = 10
	SleepAdd         = 0

	_ = "enumgen::ThreadCtrl_Sleep:Sleep"
)

// Ctrl controls a C thread.
type Ctrl struct {
	c *C.ThreadCtrl
}

// Stopper returns a Stopper via C.ThreadCtrl.
func (ctrl *Ctrl) Stopper() Stopper {
	return (*ctrlStopper)(ctrl.c)
}

// ThreadLoadStat reads LoadStat counters.
func (ctrl *Ctrl) ThreadLoadStat() (s LoadStat) {
	s.EmptyPolls = uint64(ctrl.c.nPolls[0])
	s.ValidPolls = uint64(ctrl.c.nPolls[1])
	s.Items = uint64(ctrl.c.items)
	// don't populate s.ItemsPerPoll, because Items and ValidPolls can possibly wraparound
	return s
}

// InitCtrl initializes Ctrl from *C.ThreadCtrl pointer.
func InitCtrl(ptr unsafe.Pointer) (ctrl *Ctrl) {
	ctrl = &Ctrl{
		c: (*C.ThreadCtrl)(ptr),
	}
	C.ThreadCtrl_Init(ctrl.c)
	return ctrl
}

type ctrlStopper C.ThreadCtrl

func (s *ctrlStopper) BeforeWait() {
	C.ThreadCtrl_RequestStop((*C.ThreadCtrl)(s))
}

func (s *ctrlStopper) AfterWait() {
	C.ThreadCtrl_FinishStop((*C.ThreadCtrl)(s))
}

// ThreadWithCtrl is a Thread with *Ctrl object.
type ThreadWithCtrl interface {
	ThreadWithLoadStat
	ThreadCtrl() *Ctrl
}

// NewThreadWithCtrl creates a ThreadWithCtrl.
func NewThreadWithCtrl(main cptr.Function, ctrl unsafe.Pointer) ThreadWithCtrl {
	th := &threadCtrlImpl{
		Ctrl: InitCtrl(ctrl),
	}
	th.Thread = New(main, th.Ctrl.Stopper())
	return th
}

type threadCtrlImpl struct {
	Thread
	*Ctrl
}

func (th *threadCtrlImpl) ThreadCtrl() *Ctrl {
	return th.Ctrl
}
