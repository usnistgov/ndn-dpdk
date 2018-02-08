package cs

/*
#include "../pcct/cs-entry.h"
*/
import "C"
import "unsafe"

type Entry struct {
	c *C.CsEntry
}

func EntryFromPtr(ptr unsafe.Pointer) Entry {
	return Entry{(*C.CsEntry)(ptr)}
}
