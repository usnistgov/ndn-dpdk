package dpdk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrno(t *testing.T) {
	setErrno(0x19)
	errno := GetErrno()
	assert.EqualValues(t, 0x19, errno)
}
