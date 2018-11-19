package urcu_test

import (
	"sync"
	"testing"
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestReadSide(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	rs := urcu.NewReadSide()
	defer rs.Close()

	assert.Implements((*sync.Locker)(nil), rs)
}

func TestPointer(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	rs := urcu.NewReadSide()
	defer rs.Close()

	var bare unsafe.Pointer
	bare = unsafe.Pointer(uintptr(1))

	p := urcu.NewPointer(&bare)
	v := p.Read(rs)
	assert.Equal(uintptr(1), uintptr(v))

	v = p.Xchg(unsafe.Pointer(uintptr(2)))
	assert.Equal(uintptr(1), uintptr(v))

	v = p.Read(rs)
	assert.Equal(uintptr(2), uintptr(v))
}
