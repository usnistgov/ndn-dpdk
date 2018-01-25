package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestMbuf(t *testing.T) {
	assert, _ := makeAR(t)

	var m dpdk.Mbuf
	assert.Implements((*dpdk.IMbuf)(nil), m)
}
