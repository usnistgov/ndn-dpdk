package pit

/*
#include "../pcct/pit-entry.h"
*/
import "C"
import "unsafe"

type Entry struct {
	c *C.PitEntry
}

func EntryFromPtr(ptr unsafe.Pointer) Entry {
	return Entry{(*C.PitEntry)(ptr)}
}

// Implements cs.iPitEntry.
func (entry Entry) GetPitEntryPtr() unsafe.Pointer {
	return unsafe.Pointer(entry.c)
}
