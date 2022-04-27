// Package macaddr validates and classifies MAC-48 addresses.
package macaddr

import (
	"bytes"
	"math/rand"
	"net"
)

// Equal determines whether two HardwareAddrs are the same.
func Equal(a, b net.HardwareAddr) bool {
	return bytes.Equal([]byte(a), []byte(b))
}

// IsUnicast determines whether the HardwareAddr is a non-zero unicast MAC-48 address.
func IsUnicast(a net.HardwareAddr) bool {
	return len(a) == 6 && a[0]&0x01 == 0 && a[0]|a[1]|a[2]|a[3]|a[4]|a[5] != 0
}

// IsMulticast determines whether the HardwareAddr is a multicast MAC-48 address.
func IsMulticast(a net.HardwareAddr) bool {
	return len(a) == 6 && a[0]&0x01 != 0
}

// MakeRandom generates a random unicast MAC-48 address.
func MakeRandomUnicast() (a net.HardwareAddr) {
	a = make(net.HardwareAddr, 6)
	rand.Read(a)
	a[0] |= 0x02
	a[0] &^= 0x01
	return a
}
