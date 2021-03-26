package cptr

import (
	"reflect"
	"unsafe"

	"inet.af/netstack/gohacks"
)

// AsByteSlice converts *[n]C.uint8_t or *[n]C.char or []C.uint8_t or []C.char to []byte.
func AsByteSlice(value interface{}) (b []byte) {
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Slice:
	case reflect.Array:
		panic("cannot use array value; pass pointer to array instead")
	case reflect.Ptr:
		val = val.Elem()
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
		default:
			panic(val.Type().String() + " is not an array or slice")
		}
	}

	if typ := val.Type(); typ.Elem().Size() != 1 {
		panic(typ.String() + " is incompatible with []byte")
	}

	count := val.Len()
	if count == 0 {
		return nil
	}

	sh := (*gohacks.SliceHeader)(unsafe.Pointer(&b))
	sh.Data = unsafe.Pointer(val.Index(0).UnsafeAddr())
	sh.Len = count
	sh.Cap = count
	return b
}
