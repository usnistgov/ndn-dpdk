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

// Determine whether two Entry instances point to the same underlying entry.
func (entry Entry) SameAs(entry2 Entry) bool {
	return entry.c == entry2.c
}
