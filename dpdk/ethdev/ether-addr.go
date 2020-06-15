package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"unsafe"
)

// MakeEtherAddr converts net.HardwareAddr to EtherAddr.
func MakeEtherAddr(hw net.HardwareAddr) (a EtherAddr, e error) {
	if len(hw) != C.RTE_ETHER_ADDR_LEN {
		return a, errors.New("not a MAC-48 address")
	}
	copy(a.Bytes[:], []uint8(hw))
	return a, nil
}

// ParseEtherAddr parses EtherAddr from string.
func ParseEtherAddr(input string) (a EtherAddr, e error) {
	hw, e := net.ParseMAC(input)
	if e != nil {
		return a, e
	}
	return MakeEtherAddr(hw)
}

// EtherAddr returns *C.struct_rte_ether_addr pointer.
func (a *EtherAddr) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(a)
}

func (a *EtherAddr) getPtr() *C.struct_rte_ether_addr {
	return (*C.struct_rte_ether_addr)(a.GetPtr())
}

// CopyToC copies to *C.struct_rte_ether_addr or other 6-octet buffer.
func (a EtherAddr) CopyToC(ptr unsafe.Pointer) {
	dst := (*EtherAddr)(ptr)
	*dst = a
}

// IsZero returns true if this is the zero address.
func (a EtherAddr) IsZero() bool {
	return C.rte_is_zero_ether_addr(a.getPtr()) != 0
}

// IsUnicast returns true if this is a valid assigned unicast address.
func (a EtherAddr) IsUnicast() bool {
	return C.rte_is_valid_assigned_ether_addr(a.getPtr()) != 0
}

// IsGroup returns true if this is a group address.
func (a EtherAddr) IsGroup() bool {
	return C.rte_is_multicast_ether_addr(a.getPtr()) != 0
}

// Equal determines whether two addresses are the same.
func (a EtherAddr) Equal(other EtherAddr) bool {
	return bytes.Equal(a.Bytes[:], other.Bytes[:])
}

// HardwareAddr converts EtherAddr to net.HardwareAddr.
func (a EtherAddr) HardwareAddr() net.HardwareAddr {
	return net.HardwareAddr(a.Bytes[:])
}

// String converts EtherAddr to string.
func (a EtherAddr) String() string {
	return a.HardwareAddr().String()
}

// MarshalJSON implements json.Marshaler interface.
func (a EtherAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (a *EtherAddr) UnmarshalJSON(data []byte) (e error) {
	var s string
	if e := json.Unmarshal(data, &s); e != nil {
		return e
	}
	*a, e = ParseEtherAddr(s)
	return e
}
