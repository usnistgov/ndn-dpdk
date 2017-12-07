package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_config.h>
#include <rte_mbuf.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Mbuf struct {
	ptr *C.struct_rte_mbuf
	// DO NOT add other fields: *Mbuf is passed to C code as rte_mbuf**
}

func (m Mbuf) Close() {
	C.rte_pktmbuf_free(m.ptr)
}

func (m Mbuf) GetDataLength() uint {
	return uint(m.ptr.data_len)
}

func (m Mbuf) GetHeadroom() uint {
	return uint(C.rte_pktmbuf_headroom(m.ptr))
}

func (m Mbuf) SetHeadroom(headroom uint) error {
	if m.GetDataLength() > 0 {
		return errors.New("cannot change headroom of non-empty mbuf")
	}
	if C.uint16_t(headroom) > m.ptr.buf_len {
		return errors.New("headroom cannot exceed buffer length")
	}
	m.ptr.data_off = C.uint16_t(headroom)
	return nil
}

func (m Mbuf) GetTailroom() uint {
	return uint(C.rte_pktmbuf_tailroom(m.ptr))
}

func (m Mbuf) Read(offset uint, len uint, buf unsafe.Pointer) unsafe.Pointer {
	return C.rte_pktmbuf_read(m.ptr, C.uint32_t(offset), C.uint32_t(len), buf)
}

// Prepend len octets at head, return pointer to new space.
func (m Mbuf) Prepend(len uint) (unsafe.Pointer, error) {
	res := C.rte_pktmbuf_prepend(m.ptr, C.uint16_t(len))
	if res == nil {
		return nil, errors.New("Mbuf.Prepend failed")
	}
	return unsafe.Pointer(res), nil
}

// Remove len octets from head, return pointer to new head.
func (m Mbuf) Adj(len uint) (unsafe.Pointer, error) {
	res := C.rte_pktmbuf_adj(m.ptr, C.uint16_t(len))
	if res == nil {
		return nil, errors.New("Mbuf.Adj failed")
	}
	return unsafe.Pointer(res), nil
}

// Append len octets at tail, return pointer to new space.
func (m Mbuf) Append(len uint) (unsafe.Pointer, error) {
	res := C.rte_pktmbuf_append(m.ptr, C.uint16_t(len))
	if res == nil {
		return nil, errors.New("Mbuf.Append failed")
	}
	return unsafe.Pointer(res), nil
}

// Remove len octets from tail.
func (m Mbuf) Trim(len uint) error {
	res := C.rte_pktmbuf_trim(m.ptr, C.uint16_t(len))
	if res < 0 {
		return errors.New("Mbuf.Trim failed")
	}
	return nil
}
