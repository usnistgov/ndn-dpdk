package fib

/*
#include "strategy-code.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// A global function that registers CALL-able functions available to BPF program.
// Fib.LoadStrategyCode is available only if this is specified.
var RegisterStrategyFuncs func(vm unsafe.Pointer) error

// Sequence number of StrategyCode instances, for identification purpose.
var strategyCodeId int

func fromUbpfError(funcName string, errC *C.char) error {
	err := C.GoString(errC)
	C.free(unsafe.Pointer(errC))
	return fmt.Errorf("%s: %v", funcName, err)
}

// BPF program of a forwarding strategy.
type StrategyCode struct {
	c *C.StrategyCode
}

// Load a strategy BPF program from ELF object.
func (fib *Fib) LoadStrategyCode(elf []byte) (sc StrategyCode, e error) {
	if RegisterStrategyFuncs == nil {
		return sc, errors.New("fib.RegisterStrategyFuncs is empty")
	}

	return fib.makeStrategyCode(func(vm *C.struct_ubpf_vm) error {
		e = RegisterStrategyFuncs(unsafe.Pointer(vm))
		if e != nil {
			return e
		}
		var errC *C.char
		if res := C.ubpf_load_elf(vm, unsafe.Pointer(&elf[0]), C.size_t(len(elf)), &errC); res != 0 {
			return fromUbpfError("ubpf_load_elf", errC)
		}
		return nil
	})
}

// Load an empty BPF program (mainly for unit testing).
func (fib *Fib) MakeEmptyStrategy() (sc StrategyCode, e error) {
	return fib.makeStrategyCode(func(vm *C.struct_ubpf_vm) error {
		code := []uint64{0x95}
		var errC *C.char
		if res := C.ubpf_load(vm, unsafe.Pointer(&code[0]), 8, &errC); res != 0 {
			return fromUbpfError("ubpf_load", errC)
		}
		return nil
	})
}

func (fib *Fib) makeStrategyCode(load func(vm *C.struct_ubpf_vm) error) (sc StrategyCode, e error) {
	vm := C.ubpf_create()
	if vm == nil {
		return sc, errors.New("ubpf_create failed")
	}

	if e = load(vm); e != nil {
		C.ubpf_destroy(vm)
		return sc, e
	}

	var errC *C.char
	jit := C.ubpf_compile(vm, &errC)
	if jit == nil {
		C.ubpf_destroy(vm)
		return sc, fromUbpfError("ubpf_compile", errC)
	}

	if sc.c = C.StrategyCode_Alloc(fib.c); sc.c == nil {
		C.ubpf_destroy(vm)
		return sc, dpdk.GetErrno()
	}

	strategyCodeId++
	sc.c.id = C.int(strategyCodeId)
	sc.c.vm = vm
	sc.c.jit = jit
	return sc, nil
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
