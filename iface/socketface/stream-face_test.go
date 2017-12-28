package socketface

import (
	"testing"

	"ndn-dpdk/iface"
)

func TestStreamFace(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Implements((*iface.Face)(nil), new(StreamFace))
}
