package fwdp

/*
#include "strategy.h"
*/
import "C"
import (
	"encoding/base64"
	"fmt"
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

func (s *Strategy) Close() error {
	C.ubpf_destroy(s.vm)
	return nil
}

func getMulticastStrategyElf() []byte {
	const b64 = "" +
		"f0VMRgIBAQAAAAAAAAAAAAEAAAABAAAAAAAAAAAAAAAAAAAAAAAAANABAAAAAAAAAAAAAEAAAAAA" +
		"AEAABQABAL8WAAAAAAAAtwcAAAIAAABhYQAAAAAAAFUBEgACAAAAtwcAAAAAAABxYRAAAAAAABUB" +
		"DwAAAAAAtwcAAAAAAAC3CAAAAAAAAL+BAAAAAAAAVwEAAP8AAABnAQAAAQAAAHliCAAAAAAADxIA" +
		"AAAAAABpIgAAAAAAAL9hAAAAAAAAhQAAAAAAAAAHCAAAAQAAAL+BAAAAAAAAVwEAAP8AAABxYhAA" +
		"AAAAAC0S8/8AAAAAv3AAAAAAAACVAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADoAAAAA" +
		"AAIASAAAAAAAAAAAAAAAAAAAADMAAAAAAAIAsAAAAAAAAAAAAAAAAAAAAAsAAAAQAAAAAAAAAAAA" +
		"AAAAAAAAAAAAABsAAAAQAAIAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAAgAAAAMAAAAALnJlbC50" +
		"ZXh0AEZvcndhcmRJbnRlcmVzdABQcm9ncmFtAC5zdHJ0YWIALnN5bXRhYgBMQkIwXzQATEJCMF8z" +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAAAAAAAAAAAAAAAIwAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAiAEAAAAAAABBAAAAAAAAAAAA" +
		"AAAAAAAAAQAAAAAAAAAAAAAAAAAAAAUAAAABAAAABgAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAwAAA" +
		"AAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAABAAAACQAAAAAAAAAAAAAAAAAAAAAAAAB4AQAA" +
		"AAAAABAAAAAAAAAABAAAAAIAAAAIAAAAAAAAABAAAAAAAAAAKwAAAAIAAAAAAAAAAAAAAAAAAAAA" +
		"AAAAAAEAAAAAAAB4AAAAAAAAAAEAAAADAAAACAAAAAAAAAAYAAAAAAAAAA=="
	elf, e := base64.StdEncoding.DecodeString(b64)
	if e != nil {
		panic("bad base64")
	}
	return elf
}
