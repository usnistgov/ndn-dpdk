package ndni

/*
#include "../csrc/ndni/name.h"
*/
import "C"
import (
	"errors"
	"math"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"go.uber.org/zap"
)

// LNamePrefixFilterBuilder prepares vector for C.LNamePrefixFilter_Find.
type LNamePrefixFilterBuilder struct {
	prefixL []uint16
	prefixV []byte
	index   int
	offset  int
}

// Len returns number of prefixes added.
func (b *LNamePrefixFilterBuilder) Len() int {
	return b.index
}

// Append adds a name.
func (b *LNamePrefixFilterBuilder) Append(name ndn.Name) error {
	if b.index == len(b.prefixL) {
		return errors.New("prefixL is full")
	}
	nameV, _ := name.MarshalBinary()
	nameL := len(nameV)
	if b.offset+nameL > len(b.prefixV) {
		return errors.New("prefixV is full")
	}
	b.prefixL[b.index] = uint16(nameL)
	b.index++
	copy(b.prefixV[b.offset:], nameV)
	b.offset += nameL
	return nil
}

// NewLNamePrefixFilterBuilder constructs LNamePrefixFilterBuilder.
func NewLNamePrefixFilterBuilder(prefixL unsafe.Pointer, sizeL uintptr, prefixV unsafe.Pointer, sizeV uintptr) (b *LNamePrefixFilterBuilder) {
	b = &LNamePrefixFilterBuilder{
		prefixL: unsafe.Slice((*uint16)(prefixL), sizeL/2),
		prefixV: unsafe.Slice((*uint8)(prefixV), sizeV),
	}
	for i := range b.prefixL {
		b.prefixL[i] = math.MaxUint16
	}
	return b
}

// PName represents a parsed Name.
type PName C.PName

// NewPName creates PName from ndn.Name.
func NewPName(name ndn.Name) *PName {
	var lname C.LName
	var value []byte
	if len(name) > 0 {
		value, _ = name.MarshalBinary()
		lname = C.LName{length: C.uint16_t(len(value)), value: (*C.uint8_t)(C.CBytes(value))}
	}

	pname := (*C.PName)(C.malloc(C.sizeof_PName))
	if !C.PName_Parse(pname, lname) {
		logger.Panic("PName_Parse error",
			zap.Stringer("name", name),
			zap.Binary("value", value),
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
