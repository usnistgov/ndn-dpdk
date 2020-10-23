// Package ealthread provides a thread abstraction bound to an DPDK LCore.
package ealthread

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Thread represents a procedure running on an LCore.
type Thread interface {
	// LCore returns allocated lcore.
	LCore() eal.LCore

	// SetLCore assigns an lcore.
	SetLCore(lc eal.LCore)

	// IsRunning indicates whether the thread is running.
	IsRunning() bool

	// Launch launches the thread.
	Launch()

	// Stop stops the thread.
	Stop() error
}

// New creates a Thread.
func New(main cptr.Function, stop Stopper) Thread {
	return &threadImpl{
		main: main,
		stop: stop,
	}
}

type threadImpl struct {
	lc   eal.LCore
	main cptr.Function
	stop Stopper
}

func (th *threadImpl) LCore() eal.LCore {
	return th.lc
}

func (th *threadImpl) SetLCore(lc eal.LCore) {
	th.lc = lc
}

func (th *threadImpl) IsRunning() bool {
	return th.lc.Valid() && th.lc.IsBusy()
}

func (th *threadImpl) Launch() {
	if !th.lc.Valid() {
		log.WithField("th", th).Panic("lcore unassigned")
		panic("lcore unassigned")
	}
	if th.IsRunning() {
		log.WithField("lc", th.lc).Panic("lcore is busy")
	}
	th.lc.RemoteLaunch(th.main)
}

func (th *threadImpl) Stop() error {
	if !th.IsRunning() {
		return nil
	}
	th.stop.BeforeWait()
	exitCode := th.lc.Wait()
	th.stop.AfterWait()
	if exitCode != 0 {
		return fmt.Errorf("exit code %d", exitCode)
	}
	return nil
}
