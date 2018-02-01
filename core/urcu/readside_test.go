package urcu_test

import (
	"sync"
	"testing"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestReadSide(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	rs := urcu.NewReadSide()
	defer rs.Close()

	assert.Implements((*sync.Locker)(nil), rs)
}
