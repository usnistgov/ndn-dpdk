package nameset

/*
#include "nameset.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

// NameSet is an unordered set of names.
// It is suitable for small sets.
type NameSet struct {
	c *C.NameSet
}

func New() (set NameSet) {
	return NewOnNumaSocket(dpdk.NUMA_SOCKET_ANY)
}

func NewOnNumaSocket(socket dpdk.NumaSocket) (set NameSet) {
	set.c = (*C.NameSet)(dpdk.Zmalloc("NameSet", C.sizeof_NameSet, socket))
	set.c.numaSocket = C.int(socket)
	return set
}

func FromPtr(ptr unsafe.Pointer) (set NameSet) {
	set.c = (*C.NameSet)(ptr)
	return set
}

func (set NameSet) Close() error {
	C.NameSet_Close(set.c)
	dpdk.Free(set.c)
	return nil
}

func (set NameSet) Len() int {
	return int(set.c.nRecords)
}

func (set NameSet) Insert(name *ndn.Name) {
	set.InsertWithZeroUsr(name, 0)
}

func (set NameSet) InsertWithZeroUsr(name *ndn.Name, usrLen int) (index int, usr unsafe.Pointer) {
	index = set.Len()
	C.__NameSet_Insert(set.c, (C.uint16_t)(name.Size()), (*C.uint8_t)(name.GetValue().GetPtr()),
		nil, C.size_t(usrLen))
	return index, set.GetUsr(index)
}

func (set NameSet) GetName(index int) (name *ndn.Name) {
	n := C.NameSet_GetName(set.c, C.int(index))
	name, _ = ndn.NewName(ndn.TlvBytes(C.GoBytes(unsafe.Pointer(n.value), C.int(n.length))))
	return name
}

func (set NameSet) GetUsr(index int) unsafe.Pointer {
	return C.NameSet_GetUsr(set.c, C.int(index))
}

func (set NameSet) Erase(index int) {
	C.NameSet_Erase(set.c, C.int(index))
}

func (set NameSet) FindExact(name *ndn.Name) int {
	return int(C.__NameSet_FindExact(set.c, (C.uint16_t)(name.Size()),
		(*C.uint8_t)(name.GetValue().GetPtr())))
}

func (set NameSet) FindPrefix(name *ndn.Name) int {
	return int(C.__NameSet_FindPrefix(set.c, (C.uint16_t)(name.Size()),
		(*C.uint8_t)(name.GetValue().GetPtr())))
}
