package ndntestutil

import (
	"fmt"

	"github.com/stretchr/testify/assert"

	"ndn-dpdk/ndn"
)

type getNamer interface {
	GetName() *ndn.Name
}

func getName(obj interface{}) *ndn.Name {
	switch v := obj.(type) {
	case string:
		return ndn.MustParseName(v)
	case *ndn.Name:
		return v
	case getNamer:
		return v.GetName()
	}
	panic(fmt.Errorf("cannot obtain Name from %v", obj))
}

func NameEqual(a *assert.Assertions, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool {
	return a.Equal(getName(expected).String(), getName(actual).String(), msgAndArgs...)
}

func NameIsPrefix(a *assert.Assertions, prefix interface{}, name interface{}, msgAndArgs ...interface{}) bool {
	prefixN := getName(prefix)
	nameN := getName(name)
	switch prefixN.Compare(nameN) {
	case ndn.NAMECMP_LPREFIX, ndn.NAMECMP_EQUAL:
		return true
	}
	return a.Fail(fmt.Sprintf("%s should be a prefix of %s", prefixN, nameN), msgAndArgs...)
}
