package eal

/*
#include "thread.h"
*/
import "C"
import (
	"fmt"
)

// IThread represents an application thread.
type IThread interface {
	SetLCore(lc LCore) // Assign an lcore.
	GetLCore() LCore   // Return assigned lcore.
	IsRunning() bool

	Launch() error // Launch the thread.
	Stop() error   // Stop the thread.
	Close() error  // Release data structures.
}

// ThreadBase is a base class for implementing IThread.
type ThreadBase struct {
	lc LCore
}

// SetLCore assigns an lcore to execute the thread.
func (t *ThreadBase) SetLCore(lc LCore) {
	if t.IsRunning() {
		panic("cannot change lcore while running")
	}
	t.lc = lc
}

// GetLCore returns assigned lcore.
func (t *ThreadBase) GetLCore() LCore {
	return t.lc
}

// IsRunning returns true if the thread is running.
func (t *ThreadBase) IsRunning() bool {
	return t.lc.IsValid() && t.lc.IsBusy()
}

func (t *ThreadBase) mustHaveLCore() {
	if !t.lc.IsValid() {
		panic("lcore unassigned")
	}
}

// GetNumaSocket returns the NumaSocket where this thread would be running.
func (t *ThreadBase) GetNumaSocket() NumaSocket {
	t.mustHaveLCore()
	return t.lc.GetNumaSocket()
}

// LaunchImpl launches the specifies function.
func (t *ThreadBase) LaunchImpl(f func() int) error {
	t.mustHaveLCore()
	if t.IsRunning() {
		return fmt.Errorf("lcore %d is running", t.lc)
	}
	if ok := t.lc.RemoteLaunch(f); !ok {
		return fmt.Errorf("unable to launch on %d", t.lc)
	}
	return nil
}

// StopImpl signals the function to stop.
func (t *ThreadBase) StopImpl(stop IStop) error {
	t.mustHaveLCore()
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
