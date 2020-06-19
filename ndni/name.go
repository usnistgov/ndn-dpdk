package ndni

/*
#include "../csrc/ndn/name.h"
*/
import "C"
import (
	"reflect"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

// ToName converts LName to ndn.Name.
func (lname LName) ToName() (name ndn.Name) {
	var value []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	sh.Data = uintptr(unsafe.Pointer(lname.Value))
	sh.Len = int(lname.Length)
	sh.Cap = sh.Len

	e := name.UnmarshalBinary(value)
	if e != nil {
		panic(e)
	}
	return name
}

func (pname *PName) getPtr() *C.PName {
	return (*C.PName)(unsafe.Pointer(pname))
}

// NewCName constructs CName from TLV-VALUE.
func NewCName(value []byte) (cname *CName, e error) {
	cname = new(CName)
	res := C.PName_Parse(cname.P.getPtr(), C.uint32_t(len(value)), bytesToPtr(value))
	if res != 0 {
		return nil, NdnError(res)
	}
	if len(value) > 0 {
		cname.V = &value[0]
	}
	return cname, nil
}

// CNameFromName converts ndn.Name to CName.
func CNameFromName(name ndn.Name) (cname *CName) {
	value, _ := name.MarshalBinary()
	cname, _ = NewCName(value)
	return cname
}

// ToLName converts CNmae to LName.
func (cname CName) ToLName() (lname LName) {
	lname.Value = cname.V
	lname.Length = cname.P.NOctets
	return lname
}

// ToName converts CName to ndn.Name.
func (cname CName) ToName() (name ndn.Name) {
	return cname.ToLName().ToName()
}

// Compare compares two CName objects.
func (cname *CName) Compare(other *CName) int {
	return int(C.LName_Compare(*(*C.LName)(unsafe.Pointer(cname)), *(*C.LName)(unsafe.Pointer(other))))
}

// ComputePrefixHash computes hash for prefix with i components.
func (cname *CName) ComputePrefixHash(i int) uint64 {
	return uint64(C.PName_ComputePrefixHash(cname.P.getPtr(), (*C.uint8_t)(unsafe.Pointer(cname.V)), C.uint16_t(i)))
}

// ComputeHash computes hash for all components.
func (cname *CName) ComputeHash() uint64 {
	return uint64(C.PName_ComputeHash(cname.P.getPtr(), (*C.uint8_t)(unsafe.Pointer(cname.V))))
}

func bytesToPtr(b []byte) *C.uint8_t {
	if len(b) == 0 {
		return nil
	}
	return (*C.uint8_t)(unsafe.Pointer(&b[0]))
}
