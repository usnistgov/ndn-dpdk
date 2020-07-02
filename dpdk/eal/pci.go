package eal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jaypipes/ghw"
)

// PciAddress represents a PCI address.
type PciAddress struct {
	ghw.PCIAddress
}

// Valid determines whether the PCI address is valid.
func (a PciAddress) Valid() bool {
	return ghw.PCIAddressFromString(a.String()) != nil
}

// String returns the PCI address in 0000:00:01.0 format.
func (a PciAddress) String() string {
	a.normalize()
	return fmt.Sprintf("%s:%s:%s.%s", a.Domain, a.Bus, a.Slot, a.Function)
}

// MarshalText implements encoding.TextMarshaler interface.
func (a PciAddress) MarshalText() (text []byte, e error) {
	if !a.Valid() {
		return nil, ErrPciAddress
	}
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (a *PciAddress) UnmarshalText(text []byte) (e error) {
	*a, e = ParsePciAddress(string(text))
	return e
}

func (a *PciAddress) normalize() {
	a.Domain = strings.ToLower(a.Domain)
	a.Bus = strings.ToLower(a.Bus)
	a.Slot = strings.ToLower(a.Slot)
	a.Function = strings.ToLower(a.Function)
}

// ParsePciAddress parses a PCI address.
func ParsePciAddress(input string) (a PciAddress, e error) {
	parsed := ghw.PCIAddressFromString(input)
	if parsed == nil {
		return a, ErrPciAddress
	}
	a.PCIAddress = *parsed
	a.normalize()
	return a, nil
}

// MustParsePciAddress parses a PCI string, and panics on failure.
func MustParsePciAddress(input string) (a PciAddress) {
	var e error
	if a, e = ParsePciAddress(input); e != nil {
		log.Panic(e)
	}
	return a
}

// ErrPciAddress indicates the input PCI address is invalid.
var ErrPciAddress = errors.New("bad PCI address")
