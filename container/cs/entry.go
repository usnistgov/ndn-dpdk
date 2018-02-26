package cs

/*
#include "../pcct/cs-entry.h"
*/
import "C"
import "unsafe"

type Entry struct {
	c  *C.CsEntry
	cs Cs
}

func (cs Cs) EntryFromPtr(ptr unsafe.Pointer) Entry {
	return Entry{(*C.CsEntry)(ptr), cs}
}
