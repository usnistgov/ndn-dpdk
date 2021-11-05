// Package ifacetestenv provides a test fixture for a face type.
//
// The calling test case must initialize the EAL, and create two faces that are connected together.
// The fixture sends L3 packets on one face, and expects to receive them on the other face.
package ifacetestenv

import (
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"go4.org/must"
)

var makeAR = testenv.MakeAR

// Fixture runs a test that sends and receives packets between a pair of connected faces.
type Fixture struct {
	t *testing.T

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

// NewFixture creates a Fixture.
func NewFixture(t *testing.T) (fixture *Fixture) {
	ndnitestenv.MakePacketHeadroom = mbuftestenv.Headroom(pktmbuf.DefaultHeadroom + ndni.LpHeaderHeadroom)

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
	fixture.rxQueueI = fixture.preparePktQueue(fixture.rxl.InterestDemux())
	fixture.rxQueueD = fixture.preparePktQueue(fixture.rxl.DataDemux())
	fixture.rxQueueN = fixture.preparePktQueue(fixture.rxl.NackDemux())
	fixture.txl = iface.NewTxLoop(eal.NumaSocket{})
	require.NoError(ealthread.AllocLaunch(fixture.rxl))
	require.NoError(ealthread.AllocLaunch(fixture.txl))

	return fixture
}

func (fixture *Fixture) preparePktQueue(demux *iface.InputDemux) *iface.PktQueue {
	q := (*iface.PktQueue)(eal.Zmalloc("PktQueue", unsafe.Sizeof(iface.PktQueue{}), eal.NumaSocket{}))
	q.Init(iface.PktQueueConfig{}, eal.NumaSocket{})
	demux.InitFirst()
	demux.SetDest(0, q)
	return q
}

// Close releases resources.
// This automatically closes all faces and clears LCore allocation.
func (fixture *Fixture) Close() error {
	iface.CloseAll()
	eal.Free(fixture.rxQueueI)
	eal.Free(fixture.rxQueueD)
	eal.Free(fixture.rxQueueN)
	ealthread.AllocClear()
	return nil
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
	rand.Read(content)
	mp := ndnitestenv.MakeMempools()
	txAlign := fixture.txFace.TxAlign()

	for i := 0; i < fixture.TxIterations; i++ {
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
	assert.InEpsilon(fixture.TxIterations, int(txCnt.TxInterests), fixture.TxLossTolerance)
	assert.InEpsilon(fixture.TxIterations, int(txCnt.TxData), fixture.TxLossTolerance)
	assert.InEpsilon(fixture.TxIterations, int(txCnt.TxNacks), fixture.TxLossTolerance)
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

	assert.InEpsilon(fixture.TxIterations, fixture.NRxInterests, fixture.RxLossTolerance)
	assert.InEpsilon(fixture.TxIterations, fixture.NRxData, fixture.RxLossTolerance)
	assert.InEpsilon(fixture.TxIterations, fixture.NRxNacks, fixture.RxLossTolerance)
}
