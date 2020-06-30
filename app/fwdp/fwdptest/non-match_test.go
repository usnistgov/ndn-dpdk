package fwdptest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestDataWrongName(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1")
	time.Sleep(STEP_DELAY)
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest, "/B/2") // name does not match
	time.Sleep(STEP_DELAY)
	assert.Equal(0, collect1.Count())

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestDataLongerName(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1") // no CanBePrefix
	time.Sleep(STEP_DELAY)
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest, "/B/1/Z") // name has suffix
	time.Sleep(STEP_DELAY)
	assert.Equal(0, collect1.Count())

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestDataZeroFreshnessPeriod(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1", ndn.MustBeFreshFlag) // has MustBeFresh
	time.Sleep(STEP_DELAY)
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest, 0*time.Millisecond) // no FreshnessPeriod
	time.Sleep(STEP_DELAY)
	assert.Equal(0, collect1.Count())

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestNackWrongName(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1", ndn.NonceFromUint(0xdb22330b))
	time.Sleep(STEP_DELAY)
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeNack(ndn.MakeInterest("/B/2", ndn.NonceFromUint(0xdb22330b)), collect2.Get(-1).Lp)
	time.Sleep(STEP_DELAY)
	assert.Equal(0, collect1.Count())

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NNackMiss
	}))
}

func TestNackWrongNonce(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1", ndn.NonceFromUint(0x19c3e8b8))
	time.Sleep(STEP_DELAY)
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeNack(ndn.MakeInterest("/B/1", ndn.NonceFromUint(0xf4d9aad1)), collect2.Get(-1).Lp)
	time.Sleep(STEP_DELAY)
	assert.Equal(0, collect1.Count())

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.ReadFwdInfo(i).NNackMismatch
	}))
}
