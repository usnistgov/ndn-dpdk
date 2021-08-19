// Package pciaddr parses and validates PCI addresses.
package pciaddr

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// ErrPCIAddress indicates the input PCI address is invalid.
var ErrPCIAddress = errors.New("bad PCI address")

var rePCI = regexp.MustCompile(`^(?:([[:xdigit:]]{1,4}):)?([[:xdigit:]]{1,2}):([[:xdigit:]]{1,2})\.([[:xdigit:]])$`)

// PCIAddress represents a PCI address.
type PCIAddress struct {
	Domain   uint16
	Bus      uint8
	Slot     uint8
	Function uint8
}

// String returns the PCI address in 0000:00:01.0 format.
func (a PCIAddress) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%01x", a.Domain, a.Bus, a.Slot, a.Function)
}

// MarshalText implements encoding.TextMarshaler interface.
func (a PCIAddress) MarshalText() (text []byte, e error) {
	if a.Function > 0x0F {
		return nil, ErrPCIAddress
	}
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (a *PCIAddress) UnmarshalText(text []byte) (e error) {
	*a, e = Parse(string(text))
	return e
}

// Parse parses a PCI address.
func Parse(input string) (a PCIAddress, e error) {
	m := rePCI.FindStringSubmatch(input)
	if m == nil {
		return PCIAddress{}, ErrPCIAddress
	}

	if m[1] != "" {
		u, e := strconv.ParseUint(m[1], 16, 16)
		if e != nil {
			return PCIAddress{}, ErrPCIAddress
		}
		a.Domain = uint16(u)
	}

	u, e := strconv.ParseUint(m[2], 16, 8)
	if e != nil {
		return PCIAddress{}, ErrPCIAddress
	}
	a.Bus = uint8(u)

	u, e = strconv.ParseUint(m[3], 16, 8)
	if e != nil {
		return PCIAddress{}, ErrPCIAddress
	}
	a.Slot = uint8(u)

	u, e = strconv.ParseUint(m[4], 16, 4)
	if e != nil {
		return PCIAddress{}, ErrPCIAddress
	}
	a.Function = uint8(u)

	return a, nil
}

// MustParsePCIAddress parses a PCI string, and panics on failure.
func MustParse(input string) (a PCIAddress) {
	var e error
	if a, e = Parse(input); e != nil {
		panic(e)
	}
	return a
}
