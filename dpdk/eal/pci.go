package eal

/*
#include "../../csrc/core/common.h"
#include <rte_pci.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// String returns the PCI address in 0000:00:01.0 format.
func (a PciAddress) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%1x", a.Domain, a.Bus, a.Devid, a.Function)
}

// ShortString returns the PCI address in 00:01.0 format.
func (a PciAddress) ShortString() string {
	return fmt.Sprintf("%02x:%02x.%1x", a.Bus, a.Devid, a.Function)
}

// ParsePciAddress parses a PCI address.
func ParsePciAddress(input string) (a PciAddress, e error) {
	inputC := C.CString(input)
	defer C.free(unsafe.Pointer(inputC))
	var addrC C.struct_rte_pci_addr
	if res := C.rte_pci_addr_parse(inputC, &addrC); res < 0 {
		return a, Errno(-res)
	}
	a = *(*PciAddress)(unsafe.Pointer(&addrC))
	return a, nil
}

// MustParsePciAddress parses a PCI string, and panics on failure.
func MustParsePciAddress(input string) (a PciAddress) {
	var e error
	if a, e = ParsePciAddress(input); e != nil {
		panic(e)
	}
	return a
}
