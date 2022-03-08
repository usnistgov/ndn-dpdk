// Package macaddr validates and classifies MAC-48 addresses.
package macaddr

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
)

// Equal determines whether two HardwareAddrs are the same.
func Equal(a, b net.HardwareAddr) bool {
	return bytes.Equal([]byte(a), []byte(b))
}

// IsValid determines whether the HardwareAddr is a MAC-48 address.
func IsValid(a net.HardwareAddr) bool {
	return len(a) == 6
}

// IsUnicast determines whether the HardwareAddr is a non-zero unicast MAC-48 address.
func IsUnicast(a net.HardwareAddr) bool {
	return IsValid(a) && a[0]&0x01 == 0 && a[0]|a[1]|a[2]|a[3]|a[4]|a[5] != 0
}

// IsMulticast determines whether the HardwareAddr is a multicast MAC-48 address.
func IsMulticast(a net.HardwareAddr) bool {
	return IsValid(a) && a[0]&0x01 != 0
}

// FromUint64 converts uint64 to HardwareAddr.
func FromUint64(i uint64) net.HardwareAddr {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return net.HardwareAddr(b[2:])
}

// ToUint64 converts HardwareAddr to uint64.
func ToUint64(a net.HardwareAddr) uint64 {
	if !IsValid(a) {
		return 0
	}
	b := make([]byte, 8)
	copy(b[2:], a)
	return binary.BigEndian.Uint64(b)
}

// MakeRandom generates a random MAC-48 address.
func MakeRandom(multicast bool) (a net.HardwareAddr) {
	a = make(net.HardwareAddr, 6)
	rand.Read([]byte(a))
	a[0] |= 0x02
	if multicast {
		a[0] |= 0x01
	} else {
		a[0] &^= 0x01
	}
	return a
}
