package cptr

import (
	"reflect"
	"unsafe"
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

	if val.Type().Elem().Size() != 1 {
		panic(val.Type().String() + " is incompatible with []byte")
	}

	count := val.Len()
	if count == 0 {
		return nil
	}

	*(*reflect.SliceHeader)(unsafe.Pointer(&b)) = reflect.SliceHeader{
		Data: val.Index(0).UnsafeAddr(),
		Len:  count,
		Cap:  count,
	}
	return b
}
