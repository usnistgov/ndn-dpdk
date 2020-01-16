package strategycode_test

import (
	"testing"

	"ndn-dpdk/container/strategycode"
)

func TestTable(t *testing.T) {
	assert, _ := makeAR(t)

	scP := strategycode.MakeEmpty("P")
	idP := scP.GetId()
	assert.Equal("P", scP.GetName())
	ptrP := scP.GetPtr()
	assert.NotNil(ptrP)

	scQ := strategycode.MakeEmpty("Q")
	assert.NotEqual(idP, scQ.GetId())
	assert.Len(strategycode.List(), 2)

	assert.Same(scP, strategycode.Get(idP))
	assert.Same(scP, strategycode.Find("P"))
	assert.Same(scP, strategycode.FromPtr(ptrP))

	scP2 := strategycode.MakeEmpty("P")
	assert.NotEqual(idP, scP2.GetId())
	assert.Len(strategycode.List(), 3)
	assert.Same(scP, strategycode.Get(idP))

	scP2.Close()
	assert.Len(strategycode.List(), 2)

	strategycode.DestroyAll()
	assert.Len(strategycode.List(), 0)

	assert.Nil(strategycode.Get(idP))
	assert.True(strategycode.Get(idP) == nil) // reject interface with nil underlying value
	assert.Nil(strategycode.Find("P"))
	assert.Nil(strategycode.FromPtr(ptrP))
}
