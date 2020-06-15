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

// LoadDynLibs dlopens shared libraries listed in a GROUP.
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

	var libs []string
	for _, soname := range tokens {
		if strings.HasSuffix(soname, ".so") {
			libs = append(libs, soname)
		}
	}

	// Libraries in GROUP may have dependency between each other, but each
	// successful dlopen() should resolve some of the dependencies, so that
	// maximum attempts is the number of libraries in GROUP.
	for attempts := len(libs); attempts > 0; attempts-- {
		var failedLibs []string
		for _, soname := range libs {
			sonameC := C.CString(soname)
			hdl := C.dlopen(sonameC, C.RTLD_LAZY|C.RTLD_GLOBAL)
			C.free(unsafe.Pointer(sonameC))
			if hdl == nil {
				failedLibs = append(failedLibs, soname)
			}
		}
		libs = failedLibs
		if len(libs) == 0 {
			break
		}
	}

	if len(libs) > 0 {
		return fmt.Errorf("dlopen failed for %s", strings.Join(libs, " "))
	}
	return nil
}
