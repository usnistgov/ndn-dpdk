package ealtest

import (
	"testing"

	"ndn-dpdk/dpdk/eal"
)

func TestErrno(t *testing.T) {
	assert, _ := makeAR(t)

	setErrno(0x19)
	errno := eal.GetErrno()
	assert.EqualValues(0x19, errno)
}
