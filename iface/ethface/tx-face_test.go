package ethface

import (
	"testing"

	"ndn-dpdk/iface"
)

func TestTxFace(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Implements((*iface.TxFace)(nil), new(TxFace))
}
