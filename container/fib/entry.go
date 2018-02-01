package fib

/*
#include "entry.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

const MAX_NAME_LEN = int(C.FIB_ENTRY_MAX_NAME_LEN)
const MAX_NEXTHOPS = int(C.FIB_ENTRY_MAX_NEXTHOPS)

type Entry struct {
	c C.FibEntry
}

func (entry *Entry) GetName() ndn.TlvBytes {
	name := make(ndn.TlvBytes, int(entry.c.nameL))
	for i := range name {
		name[i] = byte(entry.c.nameV[i])
	}
	return name
}

func (entry *Entry) GetNComps() int {
	return int(entry.c.nComps)
}

func (entry *Entry) SetName(name ndn.TlvBytes) error {
	if len(name) > C.FIB_ENTRY_MAX_NAME_LEN {
		return fmt.Errorf("FIB entry name cannot exceed %d octets", C.FIB_ENTRY_MAX_NAME_LEN)
	}
	entry.c.nameL = C.uint16_t(len(name))
	for i, b := range name {
		entry.c.nameV[i] = C.uint8_t(b)
	}
	entry.c.nComps = C.uint8_t(name.CountElements())
	return nil
}

func (entry *Entry) GetNexthops() (nexthops []iface.FaceId) {
	nexthops = make([]iface.FaceId, int(entry.c.nNexthops))
	for i := range nexthops {
		nexthops[i] = iface.FaceId(entry.c.nexthops[i])
	}
	return nexthops
}

func (entry *Entry) SetNexthops(nexthops []iface.FaceId) error {
	if len(nexthops) > C.FIB_ENTRY_MAX_NEXTHOPS {
		return fmt.Errorf("FIB entry cannot have more than %d nexthops", C.FIB_ENTRY_MAX_NEXTHOPS)
	}
	entry.c.nNexthops = C.uint8_t(len(nexthops))
	for i, nh := range nexthops {
		entry.c.nexthops[i] = C.FaceId(nh)
	}
	return nil
}
