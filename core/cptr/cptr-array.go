// Package cptr handles C void* pointers.
package cptr

import (
	"reflect"
	"unsafe"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

// ParseCptrArray converts any slice of pointer-sized items to C void*[] type.
func ParseCptrArray(arr any) (ptr unsafe.Pointer, count int) {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		panic(v.Type().String() + " is not a slice")
	}

	len := v.Len()
	if len == 0 {
		return nil, 0
	}

	first := v.Index(0)
	if typ := first.Type(); typ.Size() != unsafe.Sizeof(unsafe.Pointer(nil)) {
		panic(typ.String() + " element size is incompatible with C pointer")
	}

	return first.Addr().UnsafePointer(), len
}
