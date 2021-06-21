package cptrtest

/*
#include <spdk/env.h>
#include <spdk/json.h>
*/
import "C"
import (
	"math/rand"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func init() {
	// As of SPDK 21.04, explicitly calling a function in libspdk_env_dpdk.so is needed to prevent a linker error.
	C.spdk_env_get_core_count()
}

func ctestSpdkJSON(t *testing.T) {
	assert, _ := makeAR(t)

	content := make([]byte, 1048576)
	rand.Read(content)

	var obj struct {
		N int    `json:"n"`
		V string `json:"v"`
	}

	e := cptr.CaptureSpdkJSON(cptr.SpdkJSONObject(func(w0 unsafe.Pointer) {
		keyN := []C.char{'n', 0}
		keyV := []C.char{'v', 0}
		valueV := []C.char{'v', 'a', 'l', 'u', 'e', 0}

		w := (*C.struct_spdk_json_write_ctx)(w0)
		C.spdk_json_write_named_int32(w, &keyN[0], -2048)
		C.spdk_json_write_named_string(w, &keyV[0], &valueV[0])
	}), &obj)
	assert.NoError(e)
	assert.Equal(obj.N, -2048)
	assert.Equal(obj.V, "value")
}
