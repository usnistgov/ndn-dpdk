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

func (entry *Entry) GetName() (name *ndn.Name) {
	nameV := make(ndn.TlvBytes, int(entry.c.nameL))
	for i := range nameV {
		nameV[i] = byte(entry.c.nameV[i])
	}
	name, _ = ndn.NewName(nameV)
	return name
}

func (entry *Entry) CountComps() int {
	return int(entry.c.nComps)
}

func entrySetName(entryC *C.FibEntry, nameV ndn.TlvBytes, nComps int) error {
	nameL := len(nameV)
	if nameL > C.FIB_ENTRY_MAX_NAME_LEN {
		return fmt.Errorf("FIB entry name cannot exceed %d octets", C.FIB_ENTRY_MAX_NAME_LEN)
	}
	entryC.nameL = C.uint16_t(nameL)
	for i, b := range nameV {
		entryC.nameV[i] = C.uint8_t(b)
	}
	entryC.nComps = C.uint8_t(nComps)
	return nil
}

func (entry *Entry) SetName(name *ndn.Name) error {
	return entrySetName(&entry.c, name.GetValue(), name.Len())
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

func (entry *Entry) GetStrategy() StrategyCode {
	return StrategyCode{entry.c.strategy}
}

func (entry *Entry) SetStrategy(sc StrategyCode) {
	entry.c.strategy = sc.c
}
