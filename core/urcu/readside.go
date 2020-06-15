package urcu

/*
#include "../../csrc/core/urcu.h"
*/
import "C"
import (
	"runtime"
)

// ReadSide represents an RCU read-side thread.
// Fields are exported so that they can be updated to reflect what C code did.
type ReadSide struct {
	IsOnline bool
	NLocks   int
}

// NewReadSide registers current thread an an RCU read-side thread.
func NewReadSide() *ReadSide {
	runtime.LockOSThread()
	C.rcu_register_thread()
	return &ReadSide{true, 0}
}

// Close unregisters current thread as an RCU read-side thread.
func (*ReadSide) Close() error {
	C.rcu_unregister_thread()
	runtime.UnlockOSThread()
	return nil
}

// Offline marks current thread offline.
func (rs *ReadSide) Offline() {
	if rs.NLocks > 0 {
		panic("cannot go offline when locked")
	}
	rs.IsOnline = false
	C.rcu_thread_offline()
}

// Online marks current thread online.
func (rs *ReadSide) Online() {
	C.rcu_thread_online()
	rs.IsOnline = true
}

// Quiescent indicates current thread is quiescent.
func (rs *ReadSide) Quiescent() {
	if rs.NLocks > 0 {
		panic("cannot go quiescent when locked")
	}
	C.rcu_quiescent_state()
}

// Lock obtains read-side lock.
func (rs *ReadSide) Lock() {
	if !rs.IsOnline {
		panic("cannot lock when offline")
	}
	rs.NLocks++
	C.rcu_read_lock()
}

// Unlock releases read-side lock.
func (rs *ReadSide) Unlock() {
	if rs.NLocks <= 0 {
		return
	}
	C.rcu_read_unlock()
	rs.NLocks--
}
