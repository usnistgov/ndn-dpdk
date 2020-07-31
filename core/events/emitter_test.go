package events_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/events"
)

func TestOnCancel(t *testing.T) {
	assert, _ := makeAR(t)

	nA, nB := 0, 0
	fA := func() { nA++ }
	fB := func() { nB++ }

	emitter := events.NewEmitter()
	cA := emitter.On(1, fA)
	cB := emitter.On(1, fB)

	emitter.EmitSync(1)
	assert.Equal(1, nA)
	assert.Equal(1, nB)

	assert.NoError(cA.Close())
	emitter.EmitSync(1)
	assert.Equal(1, nA)
	assert.Equal(2, nB)

	assert.NoError(cA.Close())
	emitter.EmitSync(1)
	assert.Equal(1, nA)
	assert.Equal(3, nB)

	assert.NoError(cB.Close())
	emitter.EmitSync(1)
	assert.Equal(1, nA)
	assert.Equal(3, nB)
}
