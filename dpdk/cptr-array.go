package dpdk

import (
	"fmt"
	"reflect"
	"unsafe"
)

var dummyPtr unsafe.Pointer

const sizeofPtr = unsafe.Sizeof(dummyPtr)

// Cast an interface{} of any slice or array type as C void*[] type.
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
