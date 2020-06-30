package fwdptest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestSequential(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A", "sequential", face1.ID, face2.ID, face3.ID)

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(0, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face2.SetDown(true)

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(2, collect3.Count())
}
