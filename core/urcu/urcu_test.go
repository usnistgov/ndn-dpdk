package urcu_test

import (
	"sync"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
)

func TestReadSide(t *testing.T) {
	assert, _ := makeAR(t)

	rs := urcu.NewReadSide()
	defer rs.Close()

	assert.Implements((*sync.Locker)(nil), rs)
}
