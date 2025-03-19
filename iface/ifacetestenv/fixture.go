// Package ifacetestenv provides a test fixture for a face type.
package ifacetestenv

import (
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"go4.org/must"
)

var makeAR = testenv.MakeAR

// ClearFacesLCores closes all faces and clears all LCore allocations.
// This may be used in test case cleanups.
func ClearFacesLCores() {
	iface.CloseAll()
	ealthread.AllocClear()
}

// Fixture runs a test that sends and receives packets between a pair of connected faces.
//
// The calling test case must create two faces that are connected together.
// The fixture sends L3 packets on one face, and expects to receive them on the other face.
type Fixture struct {
	t testing.TB

	PayloadLen      int     // Data payload length
	DataFrames      int     // expected number of LpPackets per Data packet
	TxIterations    int     // number of TX iterations
	TxLossTolerance float64 // permitted TX packet loss (counter discrepancy)
	RxLossTolerance float64 // permitted RX packet loss

	rxl      iface.RxLoop
	rxQueueI *iface.PktQueue
	rxQueueD *iface.PktQueue
	rxQueueN *iface.PktQueue
	txl      iface.TxLoop

	rxFace iface.Face
	txFace iface.Face

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

func (fixture *Fixture) preparePktQueue(demux *iface.InputDemux) *iface.PktQueue {
	q := eal.Zmalloc[iface.PktQueue]("PktQueue", unsafe.Sizeof(iface.PktQueue{}), eal.NumaSocket{})
	q.Init(iface.PktQueueConfig{}, eal.NumaSocket{})
	demux.InitFirst()
	demux.SetDest(0, q)
	return q
}

func (fixture *Fixture) close() {
	ClearFacesLCores()
	for _, q := range []*iface.PktQueue{fixture.rxQueueI, fixture.rxQueueD, fixture.rxQueueN} {
		q.Close()
		eal.Free(q)
	}
}

// RunTest runs the test.
func (fixture *Fixture) RunTest(txFace, rxFace iface.Face) {
	fixture.rxFace = rxFace
	fixture.txFace = txFace
	CheckLocatorMarshal(fixture.t, rxFace.Locator())
	CheckLocatorMarshal(fixture.t, txFace.Locator())

	recvStop := ealthread.NewStopChan()
	go fixture.recvProc(recvStop)
	fixture.sendProc()
	time.Sleep(800 * time.Millisecond)
	recvStop.RequestStop()
	time.Sleep(100 * time.Millisecond)
}

func (fixture *Fixture) recvProc(recvStop ealthread.StopChan) {
	vec := make(pktmbuf.Vector, iface.MaxBurstSize)
	for recvStop.Continue() {
		now := eal.TscNow()
		count, _ := fixture.rxQueueI.Pop(vec, now)
		for _, pkt := range vec[:count] {
			fixture.NRxInterests += fixture.recvCheck(pkt)
		}
		count, _ = fixture.rxQueueD.Pop(vec, now)
		for _, pkt := range vec[:count] {
			fixture.NRxData += fixture.recvCheck(pkt)
		}
		count, _ = fixture.rxQueueN.Pop(vec, now)
		for _, pkt := range vec[:count] {
			fixture.NRxNacks += fixture.recvCheck(pkt)
		}
	}
}

func (fixture *Fixture) recvCheck(pkt *pktmbuf.Packet) (increment int) {
	assert, _ := makeAR(fixture.t)
	faceID := iface.ID(pkt.Port())
	assert.Equal(fixture.rxFace.ID(), faceID)
	assert.NotZero(pkt.Timestamp())
	increment = 1
	must.Close(pkt)
	return increment
}

func (fixture *Fixture) sendProc() {
	content := make([]byte, fixture.PayloadLen)
	testenv.RandBytes(content)
	mp := ndnitestenv.MakeMempools()
	txAlign := fixture.txFace.TxAlign()

	for range fixture.TxIterations {
		pkts := make([]*ndni.Packet, 3)
		pkts[0] = ndnitestenv.MakeInterest("/A")
		pkts[1] = ndnitestenv.MakeData("/A", content)
		pkts[2] = ndnitestenv.MakeNack(ndn.MakeInterest("/A"), an.NackNoRoute)
		if txAlign.Linearize {
			data := pkts[1]
			pkts[1] = data.Clone(mp, txAlign)
			data.Close()
		}
		iface.TxBurst(fixture.txFace.ID(), pkts)
		time.Sleep(time.Millisecond)
	}
}

// CheckCounters checks the counters are within acceptable range.
func (fixture *Fixture) CheckCounters() {
	assert, _ := makeAR(fixture.t)

	txCnt := fixture.txFace.Counters()
	testenv.AtOrBelow(assert, fixture.TxIterations, txCnt.TxInterests, fixture.TxLossTolerance)
	testenv.AtOrBelow(assert, fixture.TxIterations, txCnt.TxData, fixture.TxLossTolerance)
	testenv.AtOrBelow(assert, fixture.TxIterations, txCnt.TxNacks, fixture.TxLossTolerance)
	assert.InEpsilon(txCnt.TxInterests+uint64(fixture.DataFrames)*txCnt.TxData+txCnt.TxNacks, txCnt.TxFrames, 0.01)
	if fixture.DataFrames > 1 {
		assert.InEpsilon(txCnt.TxData, txCnt.TxFragGood, 0.01)
	} else {
		assert.Zero(txCnt.TxFragGood)
	}

	rxCnt := fixture.rxFace.Counters()
	assert.EqualValues(fixture.NRxInterests, rxCnt.RxInterests)
	assert.EqualValues(fixture.NRxData, rxCnt.RxData)
	assert.EqualValues(fixture.NRxNacks, rxCnt.RxNacks)
	assert.InEpsilon(rxCnt.RxInterests+uint64(fixture.DataFrames)*rxCnt.RxData+rxCnt.RxNacks, rxCnt.RxFrames, 0.01)
	if fixture.DataFrames > 1 {
		assert.InEpsilon(rxCnt.RxData, rxCnt.RxReassPackets, 0.01)
	} else {
		assert.Zero(rxCnt.RxReassPackets)
	}

	testenv.AtOrBelow(assert, fixture.TxIterations, fixture.NRxInterests, fixture.RxLossTolerance)
	testenv.AtOrBelow(assert, fixture.TxIterations, fixture.NRxData, fixture.RxLossTolerance)
	testenv.AtOrBelow(assert, fixture.TxIterations, fixture.NRxNacks, fixture.RxLossTolerance)
}

// NewFixture creates a Fixture.
func NewFixture(t testing.TB) (fixture *Fixture) {
	_, require := makeAR(t)
	fixture = &Fixture{
		t:               t,
		PayloadLen:      100,
		DataFrames:      1,
		TxIterations:    5000,
		TxLossTolerance: 0.05,
		RxLossTolerance: 0.10,
	}

	fixture.rxl = iface.NewRxLoop(eal.NumaSocket{})
	fixture.rxQueueI = fixture.preparePktQueue(fixture.rxl.DemuxOf(ndni.PktInterest))
	fixture.rxQueueD = fixture.preparePktQueue(fixture.rxl.DemuxOf(ndni.PktData))
	fixture.rxQueueN = fixture.preparePktQueue(fixture.rxl.DemuxOf(ndni.PktNack))
	fixture.txl = iface.NewTxLoop(eal.NumaSocket{})
	require.NoError(ealthread.AllocLaunch(fixture.rxl))
	require.NoError(ealthread.AllocLaunch(fixture.txl))
	time.Sleep(time.Second)

	t.Cleanup(fixture.close)
	return fixture
}
