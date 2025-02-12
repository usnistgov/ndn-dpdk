package ealthread

import (
	"time"
)

// Stopper abstracts how to tell a thread to stop.
type Stopper interface {
	// BeforeWait is invoked before lcore.Wait().
	BeforeWait()

	// AfterWait is invoked after lcore.Wait().
	AfterWait()
}

// StopChan stops a thread by sending to a channel.
type StopChan chan bool

// Continue returns true if the thread should continue.
// This should be invoked within the running thread.
func (stop StopChan) Continue() bool {
	if sleepEnabled {
		time.Sleep(time.Microsecond)
	}

	select {
	case <-stop:
		return false
	default:
		return true
	}
}

// BeforeWait requests a stop.
func (stop StopChan) BeforeWait() {
	stop <- true
}

// AfterWait completes a stop request.
func (stop StopChan) AfterWait() {
}

// RequestStop requests a stop.
//
// This may be used independently from Thread.
func (stop StopChan) RequestStop() {
	stop <- true
}

// NewStopChan constructs a StopChan.
func NewStopChan() (stop StopChan) {
	return make(StopChan)
}
