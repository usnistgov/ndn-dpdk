package dpdk

/*
#include "ethdev.h"
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"unsafe"
)

// Convert net.HardwareAddr to EtherAddr.
func MakeEtherAddr(hw net.HardwareAddr) (a EtherAddr, e error) {
	if len(hw) != C.RTE_ETHER_ADDR_LEN {
		return a, errors.New("not a MAC-48 address")
	}
	copy(a.Bytes[:], []uint8(hw))
	return a, nil
}

// Parse EtherAddr from string.
func ParseEtherAddr(input string) (a EtherAddr, e error) {
	hw, e := net.ParseMAC(input)
	if e != nil {
		return a, e
	}
	return MakeEtherAddr(hw)
}

func (a *EtherAddr) getPtr() *C.struct_rte_ether_addr {
	return (*C.struct_rte_ether_addr)(unsafe.Pointer(a))
}

// Copy to *C.struct_rte_ether_addr or other 6-octet buffer.
func (a EtherAddr) CopyToC(ptr unsafe.Pointer) {
	dst := (*EtherAddr)(ptr)
	*dst = a
}

// Determine whether this is the zero address.
func (a EtherAddr) IsZero() bool {
	return C.rte_is_zero_ether_addr(a.getPtr()) != 0
}

// Determine whether this is a valid assigned unicast address.
func (a EtherAddr) IsUnicast() bool {
	return C.rte_is_valid_assigned_ether_addr(a.getPtr()) != 0
}

// Determine whether this is a group address.
func (a EtherAddr) IsGroup() bool {
	return C.rte_is_multicast_ether_addr(a.getPtr()) != 0
}

// Determine whether two addresses are the same.
func (a EtherAddr) Equal(other EtherAddr) bool {
	return bytes.Equal(a.Bytes[:], other.Bytes[:])
}

// Convert to net.HardwareAddr.
func (a EtherAddr) HardwareAddr() net.HardwareAddr {
	return net.HardwareAddr(a.Bytes[:])
}

// Convert to string.
func (a EtherAddr) String() string {
	return a.HardwareAddr().String()
}

// Convert to JSON.
func (a EtherAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// Convert from JSON.
func (a *EtherAddr) UnmarshalJSON(data []byte) (e error) {
	var s string
	if e := json.Unmarshal(data, &s); e != nil {
		return e
	}
	*a, e = ParseEtherAddr(s)
	return e
}
