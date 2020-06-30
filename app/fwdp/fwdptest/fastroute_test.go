package fwdptest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestFastroute(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3, collect4 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3), intface.Collect(face4)
	fixture.SetFibEntry("/A/B", "fastroute", face1.ID, face2.ID, face3.ID)

	// multicast first Interest
	face4.Tx <- ndn.MakeInterest("/A/B/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	// face3 replies Data
	face3.Tx <- ndn.MakeData(collect3.Get(-1).Interest)
	fixture.StepDelay()

	// unicast to face3
	face4.Tx <- ndn.MakeInterest("/A/B/2")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(2, collect3.Count())

	// unicast to face3
	face4.Tx <- ndn.MakeInterest("/A/B/3")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(3, collect3.Count())

	// face3 fails
	face3.SetDown(true)

	// multicast next Interest because face3 failed
	face4.Tx <- ndn.MakeInterest("/A/B/4")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(2, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN

	// face1 replies Data
	face1.Tx <- ndn.MakeData(collect1.Get(-1).Interest)
	fixture.StepDelay()

	// unicast to face1
	face4.Tx <- ndn.MakeInterest("/A/B/5", ndn.NonceFromUint(0x422e9f49))
	fixture.StepDelay()
	assert.Equal(3, collect1.Count())
	assert.Equal(2, collect2.Count())
	assert.Equal(3, collect3.Count())

	// face1 replies Nack~NoRoute, retry on other faces
	face1.Tx <- ndn.MakeNack(collect1.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(3, collect1.Count())
	assert.Equal(3, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN

	// face2 replies Nack~NoRoute as well, return Nack to downstream
	collect4.Clear()
	face2.Tx <- ndn.MakeNack(collect2.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(1, collect4.Count())
	assert.NotNil(collect4.Get(-1).Nack)

	// multicast next Interest because faces Nacked
	face4.Tx <- ndn.MakeInterest("/A/B/6")
	fixture.StepDelay()
	assert.Equal(4, collect1.Count())
	assert.Equal(4, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN
}
