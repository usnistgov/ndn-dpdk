package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestErrno(t *testing.T) {
	assert, _ := makeAR(t)

	setErrno(0x19)
	errno := dpdk.GetErrno()
	assert.EqualValues(0x19, errno)
}
