// Package ealthread provides a thread abstraction bound to an DPDK LCore.
package ealthread

import (
	"errors"
	"fmt"
	"sync"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// ErrRunning indicates an error condition when a function expects the thread to be stopped.
var ErrRunning = errors.New("operation not permitted when thread is running")

var logger = logging.New("ealthread")

var activeThread sync.Map // map[eal.LCore]Thread

// Thread represents a procedure running on an LCore.
type Thread interface {
	// LCore returns allocated lcore.
	LCore() eal.LCore

	// SetLCore assigns an lcore.
	// This can only be used when the thread is stopped.
	SetLCore(lc eal.LCore)

	// IsRunning indicates whether the thread is running.
	IsRunning() bool

	// Stop stops the thread.
	Stop() error

	threadImpl() *threadImpl
}

// New creates a Thread.
func New(main cptr.Function, stop Stopper) Thread {
	return &threadImpl{
		main:    main,
		stop:    stop,
		stopped: make(chan struct{}),
	}
}

type threadImpl struct {
	lc      eal.LCore
	main    cptr.Function
	stop    Stopper
	stopped chan struct{}
}

func (th *threadImpl) LCore() eal.LCore {
	return th.lc
}

func (th *threadImpl) SetLCore(lc eal.LCore) {
	if th.IsRunning() {
		panic(ErrRunning)
	}
	th.lc = lc
}

func (th *threadImpl) IsRunning() bool {
	return th.lc.Valid() && th.lc.IsBusy()
}

func (th *threadImpl) Stop() error {
	if !th.IsRunning() {
		return nil
	}
	defer func(lc eal.LCore) { activeThread.Delete(lc) }(th.lc)
	th.stop.BeforeWait()
	exitCode := th.lc.Wait()
	th.stop.AfterWait()
	close(th.stopped)
	if exitCode != 0 {
		return fmt.Errorf("exit code %d", exitCode)
	}
	return nil
}

func (th *threadImpl) threadImpl() *threadImpl {
	return th
}

// Launch launches the thread.
func Launch(thread Thread) {
	th := thread.threadImpl()
	if !th.lc.Valid() {
		logger.Panic("lcore unassigned")
	}
	if th.IsRunning() {
		logger.Panic("lcore is busy", th.lc.ZapField("lc"))
	}
	th.stopped = make(chan struct{})
	activeThread.Store(th.lc, thread)
	th.lc.RemoteLaunch(th.main)
}
