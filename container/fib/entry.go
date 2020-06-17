package fib

/*
#include "../../csrc/fib/entry.h"

void**
FibEntry_GetStrategyPtr_(FibEntry* entry)
{
	assert(entry->maxDepth == 0);
	return (void**)&entry->strategy;
}
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

const MAX_NAME_LEN = int(C.FIB_ENTRY_MAX_NAME_LEN)
const MAX_NEXTHOPS = int(C.FIB_ENTRY_MAX_NEXTHOPS)

type Entry struct {
	c C.FibEntry
}

func entryFromC(c *C.FibEntry) *Entry {
	if c == nil {
		return nil
	}
	return &Entry{*c}
}

func (entry *Entry) GetSeqNum() uint32 {
	return uint32(entry.c.seqNum)
}

func (entry *Entry) GetName() (name ndn.Name) {
	nameV := make([]byte, int(entry.c.nameL))
	for i := range nameV {
		nameV[i] = byte(entry.c.nameV[i])
	}
	name.UnmarshalBinary(nameV)
	return name
}

func (entry *Entry) CountComps() int {
	return int(entry.c.nComps)
}

func entrySetName(entryC *C.FibEntry, name ndn.Name) error {
	nameV, _ := name.MarshalBinary()
	if len(nameV) > C.FIB_ENTRY_MAX_NAME_LEN {
		return fmt.Errorf("FIB entry name cannot exceed %d octets", C.FIB_ENTRY_MAX_NAME_LEN)
	}
	entryC.nameL = C.uint16_t(len(nameV))
	for i, b := range nameV {
		entryC.nameV[i] = C.uint8_t(b)
	}
	entryC.nComps = C.uint8_t(len(name))
	return nil
}

func (entry *Entry) SetName(name ndn.Name) error {
	return entrySetName(&entry.c, name)
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

func (entry *Entry) GetStrategy() strategycode.StrategyCode {
	return strategycode.FromPtr(*(C.FibEntry_GetStrategyPtr_(&entry.c)))
}

func (entry *Entry) SetStrategy(sc strategycode.StrategyCode) {
	*(C.FibEntry_GetStrategyPtr_(&entry.c)) = sc.GetPtr()
}
