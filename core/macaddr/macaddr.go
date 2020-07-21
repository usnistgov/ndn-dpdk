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

// IsValid determines whether the HardwareAddr is a MAC-48 address.
func IsValid(a net.HardwareAddr) bool {
	return len(a) == 6
}

// IsUnicast determines whether the HardwareAddr is a non-zero unicast MAC-48 address.
func IsUnicast(a net.HardwareAddr) bool {
	return IsValid(a) && (a[0]&0x01) == 0 && (a[0]|a[1]|a[2]|a[3]|a[4]|a[5]) != 0
}

// IsMulticast determines whether the HardwareAddr is a multicast MAC-48 address.
func IsMulticast(a net.HardwareAddr) bool {
	return IsValid(a) && (a[0]&0x01) != 0
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
