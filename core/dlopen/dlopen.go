// Package dlopen allows preloading dynamic libraries.
package dlopen

/*
#include <dlfcn.h>
#include <stdlib.h>

#cgo LDFLAGS: -ldl
*/
import "C"
import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"unsafe"
)

// Load loads a binary dynamic library.
func Load(filename string) (hdl unsafe.Pointer, e error) {
	filenameC := C.CString(filename)
	defer C.free(unsafe.Pointer(filenameC))

	C.dlerror()
	hdl = C.dlopen(filenameC, C.RTLD_LAZY|C.RTLD_GLOBAL)
	if err := C.dlerror(); err != nil {
		e = errors.New(C.GoString(err))
	}
	return
}

// LoadGroup loads a .so file that contains a GROUP.
func LoadGroup(groupFilename string) error {
	content, e := ioutil.ReadFile(groupFilename)
	if e != nil {
		return e
	}

	tokens := strings.Split(string(content), " ")
	if len(tokens) < 4 || tokens[0] != "GROUP" || tokens[1] != "(" {
		return errors.New("dlopen.LoadGroup parse error")
	}

	libs := make(groupError)
	for _, filename := range tokens {
		if strings.HasSuffix(filename, ".so") {
			libs[filename] = nil
		}
	}

	for {
		failed := make(groupError)
		for filename := range libs {
			if _, e := Load(filename); e != nil {
				failed[filename] = e
			}
		}

		switch len(failed) {
		case 0:
			return nil
		case len(libs):
			// libs in GROUP may have inter-dependencies, but each round should resolve some dependencies
			return failed
		}

		libs = failed
	}
}

type groupError map[string]error

func (e groupError) Error() string {
	var b strings.Builder
	delim := "dlopen.LoadGroup:"
	for filename, err := range e {
		fmt.Fprintf(&b, "%s %s (%v)", delim, filename, err)
		delim = ","
	}
	return b.String()
}
