package ethface

import (
	"testing"

	"ndn-dpdk/iface"
)

func TestRxFace(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Implements((*iface.RxFace)(nil), new(RxFace))
}
