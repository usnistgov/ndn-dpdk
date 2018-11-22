package appinit

/*
#include "../core/common.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// An application thread.
type IThread interface {
	fmt.Stringer

	SetLCore(lc dpdk.LCore) // Assign an LCore.
	GetLCore() dpdk.LCore   // Return assigned LCore.
	IsRunning() bool

	Launch() error // Launch the thread.
	Stop() error   // Stop the thread.
	Close() error  // Release data structures.
}

// Base class of an application thread.
type ThreadBase struct {
	lc dpdk.LCore
}

func (t *ThreadBase) ResetThreadBase() {
	t.lc = dpdk.LCORE_INVALID
}

func (t *ThreadBase) SetLCore(lc dpdk.LCore) {
	if t.lc != dpdk.LCORE_INVALID {
		panic("lcore already assigned")
	}
	t.lc = lc
}

func (t *ThreadBase) GetLCore() dpdk.LCore {
	return t.lc
}

func (t *ThreadBase) IsRunning() bool {
	return t.lc.GetState() != dpdk.LCORE_STATE_WAIT
}

func (t *ThreadBase) MustHaveLCore() {
	if t.lc == dpdk.LCORE_INVALID {
		panic("lcore unassigned")
	}
}

func (t *ThreadBase) GetNumaSocket() dpdk.NumaSocket {
	t.MustHaveLCore()
	return t.lc.GetNumaSocket()
}

func (t *ThreadBase) LaunchImpl(f dpdk.LCoreFunc) error {
	t.MustHaveLCore()
	if t.IsRunning() {
		return fmt.Errorf("lcore %d is running", t.lc)
	}
	if ok := t.lc.RemoteLaunch(f); !ok {
		return fmt.Errorf("unable to launch on %d", t.lc)
	}
	return nil
}

func (t *ThreadBase) StopImpl(stop IStop) error {
	t.MustHaveLCore()
	if !t.IsRunning() {
		return nil
	}
	stop.BeforeWait()
	exitCode := t.lc.Wait()
	stop.AfterWait()
	if exitCode != 0 {
		return fmt.Errorf("exit code %d", exitCode)
	}
	return nil
}

// Thread stop helper.
type IStop interface {
	BeforeWait() // What to do before lcore.Wait().
	AfterWait()  // What to do after lcore.Wait().
}

// Stop a thread by setting a boolean flag.
type StopFlag struct {
	c *C.bool
}

func NewStopFlag(c unsafe.Pointer) (stop StopFlag) {
	stop.c = (*C.bool)(c)
	return stop
}

func (stop StopFlag) BeforeWait() {
	*stop.c = C.bool(true)
}

func (stop StopFlag) AfterWait() {
	*stop.c = C.bool(false)
}
