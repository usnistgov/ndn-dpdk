package cptrtest

/*
#include <stdio.h>
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func ctestFileDump(t *testing.T) {
	assert, _ := makeAR(t)

	content := make([]byte, 1048576)
	randBytes(content)

	data, e := cptr.CaptureFileDump(func(fp unsafe.Pointer) {
		C.fwrite(unsafe.Pointer(&content[0]), C.size_t(len(content)), 1, (*C.FILE)(fp))
	})
	assert.NoError(e)
	assert.Equal(content, data)
}
