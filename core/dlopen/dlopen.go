package dlopen

/*
#include <dlfcn.h>
#include <stdlib.h>

#cgo LDFLAGS: -ldl
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"strings"
	"unsafe"
)

// dlopen shared libraries listed in a GROUP.
func LoadDynLibs(paths ...string) (e error) {
	var path string
	var content []byte
	for _, path = range paths {
		if content, e = ioutil.ReadFile(path); e == nil {
			break
		}
	}
	if e != nil {
		return e
	}

	tokens := strings.Split(string(content), " ")
	if len(tokens) < 4 || tokens[0] != "GROUP" || tokens[1] != "(" {
		return fmt.Errorf("unexpected text in %s", path)
	}

	for _, soname := range tokens {
		if !strings.HasSuffix(soname, ".so") {
			continue
		}
		sonameC := C.CString(soname)
		hdl := C.dlopen(sonameC, C.RTLD_LAZY)
		C.free(unsafe.Pointer(sonameC))
		if hdl == nil {
			return fmt.Errorf("dlopen failed for %s", soname)
		}
	}

	return nil
}
