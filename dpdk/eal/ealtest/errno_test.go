package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestErrno(t *testing.T) {
	assert, _ := makeAR(t)

	setErrno(9)
	errno := eal.GetErrno()
	assert.EqualValues(9, errno)

	e := eal.MakeErrno(0)
	assert.Nil(e)

	e = eal.MakeErrno(2)
	assert.NotNil(e)
	assert.EqualValues(2, e)

	e = eal.MakeErrno(-3)
	assert.NotNil(e)
	assert.EqualValues(3, e)
}
