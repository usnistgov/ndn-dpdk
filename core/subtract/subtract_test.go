package subtract_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/subtract"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var makeAR = testenv.MakeAR

type withGoodSubMethod struct {
	I    int
	nSub *int
}

func (curr withGoodSubMethod) Sub(withGoodSubMethod) withGoodSubMethod {
	*curr.nSub++
	return withGoodSubMethod{}
}

func TestGoodSubMethod(t *testing.T) {
	assert, _ := makeAR(t)
	nSubCurr, nSubPrev := 0, 0
	curr := withGoodSubMethod{I: 5, nSub: &nSubCurr}
	prev := withGoodSubMethod{I: 3, nSub: &nSubPrev}
	diff := subtract.Sub(curr, prev)
	assert.Equal(0, diff.I)
	assert.Equal(1, nSubCurr)
	assert.Equal(0, nSubPrev)
}

type withBadSubMethod struct {
	I    int
	nSub *int
}

func (curr withBadSubMethod) Sub(withBadSubMethod) int {
	*curr.nSub++
	return 0
}

func TestBadSubMethod(t *testing.T) {
	assert, _ := makeAR(t)
	nSubCurr, nSubPrev := 0, 0
	curr := withBadSubMethod{I: 5, nSub: &nSubCurr}
	prev := withBadSubMethod{I: 3, nSub: &nSubPrev}
	diff := subtract.Sub(curr, prev)
	assert.Equal(2, diff.I)
	assert.Equal(0, nSubCurr)
	assert.Equal(0, nSubPrev)
}

type structA struct {
	I int64
	U uint64
	A [2]int32
	S []uint32
	B *structB
	X int `subtract:"-"`
}

type structB struct {
	U uint
}

func TestStruct(t *testing.T) {
	assert, _ := makeAR(t)
	curr := structA{I: -5, U: 5, A: [2]int32{50, -500}, S: []uint32{5000}, B: &structB{U: 500000}, X: 5000000}
	prev := structA{I: -3, U: 3, A: [2]int32{30, -300}, S: []uint32{3000, 30000}, B: &structB{U: 300000}, X: 3000000}
	diff := subtract.Sub(curr, prev)
	assert.EqualValues(-2, diff.I)
	assert.EqualValues(2, diff.U)
	assert.Equal([2]int32{20, -200}, diff.A)
	assert.Equal([]uint32{2000}, diff.S)
	if assert.NotNil(diff.B) {
		assert.EqualValues(200000, diff.B.U)
	}
	assert.Equal(5000000, diff.X)

	zero := structA{}
	positive := subtract.Sub(curr, zero)
	assert.EqualValues(-5, positive.I)
	assert.EqualValues(5, positive.U)
	assert.Equal([2]int32{50, -500}, positive.A)
	assert.Len(positive.S, 0)
	assert.Nil(positive.B)

	negative := subtract.Sub(zero, curr)
	assert.EqualValues(5, negative.I)
	assert.EqualValues(-5, negative.U)
	assert.Equal([2]int32{-50, 500}, negative.A)
	assert.Len(negative.S, 0)
	assert.Nil(negative.B)
}
