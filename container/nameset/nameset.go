package nameset

/*
#include "nameset.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/ndn"
)

// NameSet is an unordered set of names.
// It is suitable for small sets.
type NameSet struct {
	c *C.NameSet
}

func New() (set NameSet) {
	set.c = new(C.NameSet)
	return set
}

func FromPtr(ptr unsafe.Pointer) (set NameSet) {
	set.c = (*C.NameSet)(ptr)
	return set
}

func (set NameSet) Close() error {
	C.NameSet_Close(set.c)
	return nil
}

func (set NameSet) Len() int {
	return int(set.c.nRecords)
}

func (set NameSet) Insert(comps ndn.TlvBytes) {
	C.NameSet_Insert(set.c, (*C.uint8_t)(comps.GetPtr()), C.uint16_t(len(comps)))
}

func (set NameSet) Erase(index int) {
	C.NameSet_Erase(set.c, C.int(index))
}

func (set NameSet) FindExact(comps ndn.TlvBytes) int {
	return int(C.NameSet_FindExact(set.c, (*C.uint8_t)(comps.GetPtr()), C.uint16_t(len(comps))))
}

func (set NameSet) FindPrefix(comps ndn.TlvBytes) int {
	return int(C.NameSet_FindPrefix(set.c, (*C.uint8_t)(comps.GetPtr()), C.uint16_t(len(comps))))
}
