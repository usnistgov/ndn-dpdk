package fib

/*
#include "fib.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

type Config struct {
	Id         string
	MaxEntries int
	NBuckets   int
	NumaSocket dpdk.NumaSocket
	StartDepth int
}

// The FIB.
type Fib struct {
	c          *C.Fib
	commands   chan command
	startDepth int
	nEntries   int
	nVirtuals  int
	tree       tree
}

func New(cfg Config) (fib *Fib, e error) {
	fib = new(Fib)
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	fib.c = C.Fib_New(idC, C.uint32_t(cfg.MaxEntries), C.uint32_t(cfg.NBuckets),
		C.unsigned(cfg.NumaSocket), C.uint8_t(cfg.StartDepth))
	if fib.c == nil {
		return nil, dpdk.GetErrno()
	}

	fib.startDepth = cfg.StartDepth

	fib.commands = make(chan command)
	go fib.commandLoop()

	return fib, nil
}

// Get native *C.Fib pointer to use in other packages.
func (fib *Fib) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(fib.c)
}

// Get number of FIB entries, excluding virtual entries.
func (fib *Fib) Len() int {
	return fib.nEntries
}

// Get number of virtual entries.
func (fib *Fib) CountVirtuals() int {
	return fib.nVirtuals
}

type command struct {
	f    func(rs *urcu.ReadSide) error
	done chan<- error
}

// An RCU read-side thread to execute all commands.
func (fib *Fib) commandLoop() {
	rs := urcu.NewReadSide()
	defer rs.Close()
	rs.Offline()
	for cmd, ok := <-fib.commands; ok; cmd, ok = <-fib.commands {
		rs.Online()
		cmd.done <- cmd.f(rs)
		rs.Offline()
	}
}

// Execute a command in an RCU read-side thread.
func (fib *Fib) postCommand(f func(rs *urcu.ReadSide) error) error {
	done := make(chan error)
	fib.commands <- command{f: f, done: done}
	return <-done
}

func (fib *Fib) Close() (e error) {
	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		C.Fib_Close(fib.c)
		return nil
	})
	close(fib.commands)
	return e
}
