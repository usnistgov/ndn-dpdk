package ndntestenv

import (
	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

type getNamer interface {
	GetName() *ndni.Name
}

func getName(obj interface{}) *ndni.Name {
	switch v := obj.(type) {
	case string:
		return ndni.MustParseName(v)
	case *ndni.Name:
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
	case ndni.NAMECMP_LPREFIX, ndni.NAMECMP_EQUAL:
		return true
	}
	return a.Fail(fmt.Sprintf("%s should be a prefix of %s", prefixN, nameN), msgAndArgs...)
}
