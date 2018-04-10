package fib

/*
#include "strategy-code.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
)

var strategyCodeId int

// BPF program of a forwarding strategy.
type StrategyCode struct {
	c *C.StrategyCode
}

// Allocate a StrategyCode instance from FIB's mempool.
func (fib *Fib) AllocStrategyCode() (sc StrategyCode, e error) {
	sc.c = C.StrategyCode_Alloc(fib.c)
	if sc.c == nil {
		return sc, dpdk.GetErrno()
	}

	strategyCodeId++
	sc.c.id = C.int(strategyCodeId)
	return sc, nil
}

// Assign a prepared C.struct_ubpf_vm instance.
func (sc StrategyCode) Assign(vm unsafe.Pointer) error {
	sc.c.vm = (*C.struct_ubpf_vm)(vm)

	var errC *C.char
	sc.c.jit = C.ubpf_compile(sc.c.vm, &errC)
	if sc.c.jit == nil {
		err := C.GoString(errC)
		C.free(unsafe.Pointer(errC))
		return fmt.Errorf("ubpf_compile: %v", err)
	}

	return nil
}

// Load an empty BPF program.
// This is mainly for unit testing.
func (sc StrategyCode) LoadEmpty() error {
	vm := C.ubpf_create()
	if vm == nil {
		return fmt.Errorf("ubpf_create: allocation error")
	}

	code := []uint64{0x95}
	var errC *C.char
	res := C.ubpf_load(vm, unsafe.Pointer(&code[0]), 8, &errC)
	if res != 0 {
		err := C.GoString(errC)
		C.free(unsafe.Pointer(errC))
		return fmt.Errorf("ubpf_load: %v", err)
	}

	return sc.Assign(unsafe.Pointer(vm))
}

func (sc StrategyCode) GetId() int {
	return int(sc.c.id)
}

func (sc StrategyCode) CountRefs() int {
	return int(sc.c.nRefs)
}

func (sc StrategyCode) Ref() {
	C.StrategyCode_Ref(sc.c)
}

func (sc StrategyCode) Unref() {
	C.StrategyCode_Unref(sc.c)
}
