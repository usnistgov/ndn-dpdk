// Package subtract computes struct numerical difference.
package subtract

import (
	"reflect"

	"github.com/zyedidia/generic"
)

// Sub computes the numerical difference between two structs of the same type.
// It returns the result in a new instance of the same type, and does not modify the arguments.
//
// If the struct has `func (T) Sub(T) T` method, it is used; otherwise, this calls SubFields.
func Sub[T any](curr, prev T) (diff T) {
	return subV(reflect.ValueOf(curr), reflect.ValueOf(prev)).Interface().(T)
}

func subV(currV, prevV reflect.Value) (diffV reflect.Value) {
	typ := currV.Type()
	if method, ok := typ.MethodByName("Sub"); ok &&
		method.Type.NumIn() == 2 && method.Type.NumOut() == 1 && method.Type.In(1) == typ && method.Type.Out(0) == typ {
		return method.Func.Call([]reflect.Value{currV, prevV})[0]
	}

	diffV = reflect.New(typ).Elem()
	subFieldsV(currV, prevV, diffV)
	return diffV
}

// SubFields computes the numerical difference between two structs of the same type.
// It assigns the result to a pointer to the same type.
//
// This function can handle these field types:
//  - unsigned/signed integer.
//  - struct (recursive).
//  - slice (recursive, truncated to the shorter slice).
//  - array (recursive).
// Other fields are ignored.
// A field may be explicitly skipped with `subtract:"-"` tag.
func SubFields[T any](curr, prev T, diffPtr *T) {
	subFieldsV(reflect.ValueOf(curr), reflect.ValueOf(prev), reflect.ValueOf(diffPtr).Elem())
}

func subFieldsV(currV, prevV, diffV reflect.Value) {
	for _, field := range reflect.VisibleFields(currV.Type()) {
		if !field.IsExported() || field.Tag.Get("subtract") == "-" {
			continue
		}
		subValue(currV.FieldByIndex(field.Index), prevV.FieldByIndex(field.Index), diffV.FieldByIndex(field.Index))
	}
}

func subValue(currV, prevV, diffV reflect.Value) {
	switch currV.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		diffV.SetUint(currV.Uint() - prevV.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		diffV.SetInt(currV.Int() - prevV.Int())
	case reflect.Struct:
		diffV.Set(subV(currV, prevV))
	case reflect.Slice:
		length := generic.Min(currV.Len(), prevV.Len())
		diffV.Set(reflect.MakeSlice(currV.Type(), length, length))
		fallthrough
	case reflect.Array:
		for i, length := 0, diffV.Len(); i < length; i++ {
			subValue(currV.Index(i), prevV.Index(i), diffV.Index(i))
		}
	case reflect.Ptr:
		if !currV.IsNil() && !prevV.IsNil() {
			diffV.Set(reflect.New(currV.Type().Elem()))
			subValue(currV.Elem(), prevV.Elem(), diffV.Elem())
		}
	}
}
