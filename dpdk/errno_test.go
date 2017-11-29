package dpdk

import "testing"

func TestErrno(t *testing.T) {
	setErrno(0x19)
	errno := GetErrno()
	if errno != 0x19 {
		t.Errorf("errno %d != 0x19", errno)
	}
}