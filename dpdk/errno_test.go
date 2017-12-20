package dpdk

import (
	"testing"
)

func TestErrno(t *testing.T) {
	assert, _ := makeAR(t)

	setErrno(0x19)
	errno := GetErrno()
	assert.EqualValues(0x19, errno)
}
