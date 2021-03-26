package strategycode_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"go4.org/must"
)

func TestTable(t *testing.T) {
	assert, _ := makeAR(t)

	scP := strategycode.MakeEmpty("P")
	idP := scP.ID()
	assert.Equal("P", scP.Name())
	ptrP := scP.Ptr()
	assert.NotNil(ptrP)

	scQ := strategycode.MakeEmpty("Q")
	assert.NotEqual(idP, scQ.ID())
	assert.Len(strategycode.List(), 2)

	assert.Same(scP, strategycode.Get(idP))
	assert.Same(scP, strategycode.Find("P"))
	assert.Same(scP, strategycode.FromPtr(ptrP))

	scP2 := strategycode.MakeEmpty("P")
	assert.NotEqual(idP, scP2.ID())
	assert.Len(strategycode.List(), 3)
	assert.Same(scP, strategycode.Get(idP))

	must.Close(scP2)
	assert.Len(strategycode.List(), 2)

	strategycode.DestroyAll()
	assert.Len(strategycode.List(), 0)

	assert.Nil(strategycode.Get(idP))
	assert.Nil(strategycode.Find("P"))
}
