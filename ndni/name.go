package ndni

/*
#include "../csrc/ndni/name.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"go.uber.org/zap"
)

// PNameToName converts PName to ndn.Name.
func PNameToName(pname unsafe.Pointer) (name ndn.Name) {
	p := (*C.PName)(pname)
	if p.length == 0 {
		return ndn.Name{}
	}
	value := C.GoBytes(unsafe.Pointer(p.value), C.int(p.length))
	if e := name.UnmarshalBinary(value); e != nil {
		logger.Panic("name.UnmarshalBinary error", zap.Error(e))
	}
	return name
}

// PName represents a parsed Name.
type PName C.PName

// NewPName creates PName from ndn.Name.
func NewPName(name ndn.Name) *PName {
	var lname C.LName
	if len(name) == 0 {
		lname = C.LName_Empty()
	} else {
		value, _ := name.MarshalBinary()
		valueC := C.CBytes(value)
		lname = C.LName_Init(C.uint16_t(len(value)), (*C.uint8_t)(valueC))
	}
	pname := (*C.PName)(C.malloc(C.sizeof_PName))
	ok := bool(C.PName_Parse(pname, lname))
	if !ok {
		logger.Panic("PName_Parse error",
			zap.Stringer("name", name),
		)
	}
	return (*PName)(pname)
}

// Ptr return *C.PName or *C.LName pointer.
func (p *PName) Ptr() unsafe.Pointer {
	return unsafe.Pointer(p)
}

func (p *PName) lname() C.LName {
	return *(*C.LName)(p.Ptr())
}

// Free releases memory.
func (p *PName) Free() {
	pname := (*C.PName)(p)
	if pname.value != nil {
		C.free(unsafe.Pointer(pname.value))
	}
	C.free(unsafe.Pointer(pname))
}

// ComputeHash returns LName hash.
func (p *PName) ComputeHash() uint64 {
	return uint64(C.LName_ComputeHash(p.lname()))
}
