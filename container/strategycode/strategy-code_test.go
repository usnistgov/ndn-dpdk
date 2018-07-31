package strategycode_test

import (
	"testing"

	"ndn-dpdk/container/strategycode"
)

func TestTable(t *testing.T) {
	assert, _ := makeAR(t)

	scP := strategycode.MakeEmpty("P")
	idP := scP.GetId()
	scQ := strategycode.MakeEmpty("Q")
	assert.NotEqual(idP, scQ.GetId())
	assert.Len(strategycode.List(), 2)

	if sc, ok := strategycode.Get(idP); assert.True(ok) {
		assert.Equal(idP, sc.GetId())
		assert.Equal("P", sc.GetName())
	}
	if sc, ok := strategycode.Find("P"); assert.True(ok) {
		assert.Equal(idP, sc.GetId())
	}

	scP2 := strategycode.MakeEmpty("P")
	assert.NotEqual(idP, scP2.GetId())
	assert.Len(strategycode.List(), 3)

	if sc, ok := strategycode.Get(idP); assert.True(ok) {
		assert.Equal(idP, sc.GetId())
	}

	scP2.Close()
	assert.Len(strategycode.List(), 2)

	strategycode.CloseAll()
	assert.Len(strategycode.List(), 0)

	_, ok := strategycode.Get(idP)
	assert.False(ok)
	_, ok = strategycode.Find("P")
	assert.False(ok)
}
