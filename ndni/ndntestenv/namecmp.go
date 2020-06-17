package ndntestenv

import (
	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

type getNamer interface {
	GetName() ndn.Name
}

func getName(obj interface{}) ndn.Name {
	switch v := obj.(type) {
	case string:
		return ndn.ParseName(v)
	case ndn.Name:
		return v
	case getNamer:
		return v.GetName()
	}
	panic(fmt.Errorf("cannot obtain Name from %T", obj))
}

// NameEqual asserts that actual name equals expected name.
func NameEqual(a *assert.Assertions, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool {
	return a.Equal(getName(expected).String(), getName(actual).String(), msgAndArgs...)
}

// NameIsPrefix asserts that prefix is a prefix of name.
func NameIsPrefix(a *assert.Assertions, prefix interface{}, name interface{}, msgAndArgs ...interface{}) bool {
	prefixN := getName(prefix)
	nameN := getName(name)
	if prefixN.IsPrefixOf(nameN) {
		return true
	}
	return a.Fail(fmt.Sprintf("%s should be a prefix of %s", prefixN, nameN), msgAndArgs...)
}
