package cptr

import (
	"fmt"
	"reflect"
	"unsafe"
)

const sizeofPtr = unsafe.Sizeof(unsafe.Pointer(nil))

// ParseCptrArray converts an interface{} of any slice or array type to C void*[] type.
func ParseCptrArray(arr interface{}) (ptr unsafe.Pointer, count int) {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		panic(fmt.Sprintf("%T is not a slice or array", arr))
	}

	len := v.Len()
	if len == 0 {
		return nil, 0
	}

	first := v.Index(0)
	if first.Type().Size() != sizeofPtr {
		panic(fmt.Sprintf("%T element size is incompatible with C pointer", arr))
	}

	return unsafe.Pointer(first.UnsafeAddr()), len
}
