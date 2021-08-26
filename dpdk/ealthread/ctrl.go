package ealthread

/*
#include "../../csrc/dpdk/thread.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
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
