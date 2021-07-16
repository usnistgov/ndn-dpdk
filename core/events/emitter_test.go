package events_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/events"
)

func TestOnCancel(t *testing.T) {
	assert, _ := makeAR(t)

	nA, nB, nC, nD := 0, 0, 0, 0
	fA := func() { nA++ }
	fB := func() { nB++ }
	fC := func() { nC++ }
	fD := func() { nD++ }

	emitter := events.NewEmitter()
	cancelA := emitter.On(1, fA)
	cancelB := emitter.On(1, fB)
	cancelC := emitter.Once(2, fC)
	cancelD := emitter.Once(2, fD)

	emitter.Emit(1)
	assert.Equal(1, nA)
	assert.Equal(1, nB)

	cancelA()
	emitter.Emit(1)
	assert.Equal(1, nA)
	assert.Equal(2, nB)

	cancelA()
	emitter.Emit(1)
	assert.Equal(1, nA)
	assert.Equal(3, nB)

	cancelB()
	emitter.Emit(1)
	assert.Equal(1, nA)
	assert.Equal(3, nB)

	cancelD()
	emitter.Emit(2)
	assert.Equal(1, nC)
	assert.Equal(0, nD)

	emitter.Emit(2)
	assert.Equal(1, nC)
	assert.Equal(0, nD)

	cancelC()
	emitter.Emit(2)
	assert.Equal(1, nC)
	assert.Equal(0, nD)
}
