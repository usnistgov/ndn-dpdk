// Package cptr handles C void* pointers.
package cptr

import (
	"reflect"
	"unsafe"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

const sizeofPtr = unsafe.Sizeof(unsafe.Pointer(nil))

// ParseCptrArray converts any slice of pointer-sized items to C void*[] type.
func ParseCptrArray(arr interface{}) (ptr unsafe.Pointer, count int) {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		panic(v.Type().String() + " is not a slice")
	}

	len := v.Len()
	if len == 0 {
		return nil, 0
	}

	first := v.Index(0)
	if typ := first.Type(); typ.Size() != sizeofPtr {
		panic(typ.String() + " element size is incompatible with C pointer")
	}

	return unsafe.Pointer(first.UnsafeAddr()), len
}
