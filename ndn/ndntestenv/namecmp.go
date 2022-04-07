package ndntestenv

import (
	"fmt"
	"reflect"

	"github.com/stretchr/testify/assert"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type getNamer interface {
	Name() ndn.Name
}

func getName(obj any) ndn.Name {
	switch v := obj.(type) {
	case string:
		return ndn.ParseName(v)
	case ndn.Name:
		return v
	case getNamer:
		return v.Name()
	default:
		val := reflect.Indirect(reflect.ValueOf(obj))
		return val.FieldByName("Name").Interface().(ndn.Name)
	}
}

// NameEqual asserts that actual name equals expected name.
// Name arguments can be string, Name, object with Name() method, or object with Name field.
func NameEqual(a *assert.Assertions, expected, actual any, msgAndArgs ...any) bool {
	return a.Equal(getName(expected).String(), getName(actual).String(), msgAndArgs...)
}

// NameIsPrefix asserts that name starts with prefix.
// Name arguments can be string, Name, object with Name() method, or object with Name field.
func NameIsPrefix(a *assert.Assertions, prefix, name any, msgAndArgs ...any) bool {
	prefixN := getName(prefix)
	nameN := getName(name)
	if prefixN.IsPrefixOf(nameN) {
		return true
	}
	return a.Fail(fmt.Sprintf("%s should be a prefix of %s", prefixN, nameN), msgAndArgs...)
}
