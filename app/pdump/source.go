package pdump

/*
#include "../../csrc/pdump/source.h"
*/
import "C"
import (
	"sync"
	"unsafe"

	"go.uber.org/zap"
)

var sourcesLock sync.Mutex

func setSourceRef(ref *C.PdumpSourceRef, expected, newPtr *C.PdumpSource) {
	if old := C.PdumpSourceRef_Set(ref, newPtr); old != expected {
		logger.Panic("PdumpSourceRef_Set pointer mismatch",
			zap.Uintptr("new", uintptr(unsafe.Pointer(newPtr))),
			zap.Uintptr("old", uintptr(unsafe.Pointer(old))),
			zap.Uintptr("expected", uintptr(unsafe.Pointer(expected))),
		)
	}
}
