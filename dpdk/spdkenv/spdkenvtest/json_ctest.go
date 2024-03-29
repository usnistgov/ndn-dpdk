package spdkenvtest

/*
#include <spdk/env.h>
#include <spdk/json.h>
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

var (
	makeAR    = testenv.MakeAR
	randBytes = testenv.RandBytes
)

func ctestJSON(t *testing.T) {
	assert, _ := makeAR(t)

	content := make([]byte, 1048576)
	randBytes(content)

	var obj struct {
		N int    `json:"n"`
		V string `json:"v"`
	}

	e := spdkenv.CaptureJSON(spdkenv.JSONObject(func(w0 unsafe.Pointer) {
		keyN := []C.char{'n', 0}
		keyV := []C.char{'v', 0}
		valueV := []C.char{'v', 'a', 'l', 'u', 'e', 0}

		w := (*C.struct_spdk_json_write_ctx)(w0)
		C.spdk_json_write_named_int32(w, unsafe.SliceData(keyN), -2048)
		C.spdk_json_write_named_string(w, unsafe.SliceData(keyV), unsafe.SliceData(valueV))
	}), &obj)
	assert.NoError(e)
	assert.Equal(obj.N, -2048)
	assert.Equal(obj.V, "value")
}
