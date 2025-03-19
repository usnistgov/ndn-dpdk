// Package macaddr validates MAC-48 addresses.
package macaddr

import (
	"crypto/rand"
	"errors"
	"net"
)

const (
	// MinVLAN is the minimum VLAN number.
	MinVLAN = 0x001

	// MaxVLAN is the maximum VLAN number.
	MaxVLAN = 0xFFE
)

// Error conditions.
var (
	ErrAddr      = errors.New("invalid MAC address")
	ErrUnicast   = errors.New("invalid unicast MAC address")
	ErrMulticast = errors.New("invalid multicast MAC address")
	ErrVLAN      = errors.New("invalid VLAN")
)

// IsUnicast determines whether the HardwareAddr is a non-zero unicast MAC-48 address.
func IsUnicast(a net.HardwareAddr) bool {
	return len(a) == 6 && a[0]&0x01 == 0 && a[0]|a[1]|a[2]|a[3]|a[4]|a[5] != 0
}

// IsMulticast determines whether the HardwareAddr is a multicast MAC-48 address.
func IsMulticast(a net.HardwareAddr) bool {
	return len(a) == 6 && a[0]&0x01 != 0
}

// MakeRandomUnicast generates a random unicast MAC-48 address.
func MakeRandomUnicast() (a net.HardwareAddr) {
	a = make(net.HardwareAddr, 6)
	rand.Read(a)
	a[0] |= 0x02
	a[0] &^= 0x01
	return a
}

// IsVLAN determines whether an integer is a valid VLAN identifier.
func IsVLAN(vlan int) bool {
	return vlan >= MinVLAN && vlan <= MaxVLAN
}
