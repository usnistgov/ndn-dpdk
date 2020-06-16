package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestCtx(t *testing.T) {
	assert, _ := makeAR(t)

	obj1 := 1111
	ctx1 := cptr.CtxPut(&obj1)
	assert.Equal(&obj1, cptr.CtxGet(ctx1))
	assert.Equal(&obj1, cptr.CtxPop(ctx1))
	assert.Panics(func() { cptr.CtxGet(ctx1) })

	obj2 := 2222
	ctx2 := cptr.CtxPut(&obj2)
	assert.Equal(&obj2, cptr.CtxGet(ctx2))
	cptr.CtxClear(ctx2)
	assert.Panics(func() { cptr.CtxGet(ctx2) })
}
