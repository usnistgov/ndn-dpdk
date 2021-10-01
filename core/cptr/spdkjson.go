package cptr

/*
#include "../../csrc/core/common.h"
#include <spdk/env.h>
#include <spdk/json.h>

static struct spdk_json_write_ctx*
c_spdk_json_write_begin(void* write_cb, uintptr_t cb_ctx, uint32_t flags)
{
	return spdk_json_write_begin((spdk_json_write_cb)write_cb, (void*)cb_ctx, flags);
}

int go_spdkJSONWrite(uintptr_t ctx, void* data, size_t size);
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"runtime/cgo"
	"unsafe"
)

// As of SPDK 21.07, explicitly calling a function in libspdk_env_dpdk.so is needed to prevent a linker error:
//  /usr/local/lib/libspdk_util.so: undefined reference to `spdk_realloc'
var _ = C.spdk_env_get_core_count()

// CaptureSpdkJSON invokes a function that writes to *C.struct_spdk_json_write_ctx, and unmarshals what's been written.
func CaptureSpdkJSON(f func(w unsafe.Pointer), ptr interface{}) (e error) {
	buf := new(bytes.Buffer)
	ctx := cgo.NewHandle(buf)
	defer ctx.Delete()

	w := C.c_spdk_json_write_begin(C.go_spdkJSONWrite, C.uintptr_t(ctx), 0)
	f(unsafe.Pointer(w))
	if res := C.spdk_json_write_end(w); res != 0 {
		return errors.New("spdk_json_write_end failed")
	}
	return json.Unmarshal(buf.Bytes(), ptr)
}

// SpdkJSONObject can be used with CaptureSpdkJSON to wrap the output in a JSON object.
func SpdkJSONObject(f func(w unsafe.Pointer)) func(w unsafe.Pointer) {
	return func(w unsafe.Pointer) {
		jw := (*C.struct_spdk_json_write_ctx)(w)
		C.spdk_json_write_object_begin(jw)
		f(w)
		C.spdk_json_write_object_end(jw)
	}
}

//export go_spdkJSONWrite
func go_spdkJSONWrite(ctx C.uintptr_t, data unsafe.Pointer, size C.size_t) C.int {
	buf := cgo.Handle(ctx).Value().(*bytes.Buffer)
	buf.Write(C.GoBytes(data, C.int(size)))
	return 0
}
