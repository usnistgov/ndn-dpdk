package eal

/*
#include "thread.h"
*/
import "C"
import (
	"unsafe"
)

// IStop abstracts how to tell a thread top stop.
type IStop interface {
	BeforeWait() // What to do before lcore.Wait().
	AfterWait()  // What to do after lcore.Wait().
}

// StopWait stops a thread by waiting for it indefinitely.
type StopWait struct{}

func (stop StopWait) BeforeWait() {}
func (stop StopWait) AfterWait()  {}

// StopFlag stops a thread by setting a boolean flag.
type StopFlag struct {
	c *C.ThreadStopFlag
}

// NewStopFlag constructs a StopFlag from initialized C pointer.
func NewStopFlag(c unsafe.Pointer) (stop StopFlag) {
	stop.c = (*C.ThreadStopFlag)(c)
	return stop
}

// InitStopFlag constructs and initializes a StopFlag.
func InitStopFlag(c unsafe.Pointer) (stop StopFlag) {
	stop = NewStopFlag(c)
	C.ThreadStopFlag_Init(stop.c)
	return stop
}

// BeforeWait requests a stop.
func (stop StopFlag) BeforeWait() {
	C.ThreadStopFlag_RequestStop(stop.c)
}

// AfterWait completes a stop request.
func (stop StopFlag) AfterWait() {
	C.ThreadStopFlag_FinishStop(stop.c)
}

// StopChan stops a thread by sending to a channel.
type StopChan chan bool

// NewStopChan constructs a StopChan.
func NewStopChan() (stop StopChan) {
	return make(StopChan)
}

// Continue returns true if the thread should continue.
// This should be invoked within the running thread.
func (stop StopChan) Continue() bool {
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
