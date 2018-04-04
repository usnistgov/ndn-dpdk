package fwdp

/*
#include "strategy.h"
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"unsafe"
)

// BPF code of a strategy.
type Strategy struct {
	vm  *C.struct_ubpf_vm
	jit C.ubpf_jit_fn
}

func NewStrategy(elf []byte) (s *Strategy, e error) {
	s = new(Strategy)

	if s.vm = C.ubpf_create(); s.vm == nil {
		return nil, fmt.Errorf("ubpf_create failed")
	}

	if regFuncErr := C.SgRegisterFuncs(s.vm); regFuncErr != 0 {
		return nil, fmt.Errorf("SgRegisterFuncs: %d errors", regFuncErr)
	}

	var errC *C.char
	if res := C.ubpf_load_elf(s.vm, unsafe.Pointer(&elf[0]), C.size_t(len(elf)), &errC); res != 0 {
		err := C.GoString(errC)
		C.free(unsafe.Pointer(errC))
		return nil, fmt.Errorf("ubpf_load_elf: %s", err)
	}

	if s.jit = C.ubpf_compile(s.vm, &errC); s.jit == nil {
		err := C.GoString(errC)
		C.free(unsafe.Pointer(errC))
		return nil, fmt.Errorf("ubpf_compile: %s", err)
	}

	return s, nil
}

func NewBuiltinStrategy(name string) (*Strategy, error) {
	_, codeFilePath, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("runtime.Caller failed")
	}

	objFilePath := path.Join(path.Dir(codeFilePath),
		fmt.Sprintf("../../build/strategy/%s.o", name))
	elf, e := ioutil.ReadFile(objFilePath)
	if e != nil {
		return nil, fmt.Errorf("ioutil.ReadFile(%s): %v", objFilePath, e)
	}
	return NewStrategy(elf)
}

func (s *Strategy) Close() error {
	C.ubpf_destroy(s.vm)
	return nil
}
