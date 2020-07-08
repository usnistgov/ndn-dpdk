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
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

var makeAR = testenv.MakeAR

// Fixture runs a test that sends and receives packets between a pair of connected faces.
type Fixture struct {
	t *testing.T

	PayloadLen    int     // Data payload length
	TxIterations  int     // number of TX iterations
	LossTolerance float64 // permitted packet loss

	rxl      iface.RxLoop
	rxQueueI *iface.PktQueue
	rxQueueD *iface.PktQueue
	rxQueueN *iface.PktQueue
	txl      iface.TxLoop

	rxFace    iface.Face
	rxDiscard map[iface.ID]iface.Face
	recvStop  chan bool
	txFace    iface.Face

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

// New creates a Fixture.
func New(t *testing.T) (fixture *Fixture) {
	_, require := makeAR(t)
	fixture = new(Fixture)
	fixture.t = t

	fixture.PayloadLen = 100
	fixture.TxIterations = 10000
	fixture.LossTolerance = 0.1

	fixture.rxl = iface.NewRxLoop(eal.NumaSocket{})
	fixture.rxQueueI = fixture.preparePktQueue(fixture.rxl.InterestDemux())
	fixture.rxQueueD = fixture.preparePktQueue(fixture.rxl.DataDemux())
	fixture.rxQueueN = fixture.preparePktQueue(fixture.rxl.NackDemux())
	fixture.txl = iface.NewTxLoop(eal.NumaSocket{})
	require.NoError(ealthread.Launch(fixture.rxl))
	require.NoError(ealthread.Launch(fixture.txl))

	fixture.rxDiscard = make(map[iface.ID]iface.Face)
	fixture.recvStop = make(chan bool)

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
	eal.Free(fixture.rxQueueI)
	eal.Free(fixture.rxQueueD)
	eal.Free(fixture.rxQueueN)
	iface.CloseAll()
	ealthread.DefaultAllocator.Clear()
	return nil
}

// AddRxDiscard indicates that packets received at the specified face should be dropped.
func (fixture *Fixture) AddRxDiscard(face iface.Face) {
	fixture.rxDiscard[face.ID()] = face
}

// RunTest executes the test.
func (fixture *Fixture) RunTest(txFace, rxFace iface.Face) {
	fixture.rxFace = rxFace
	fixture.txFace = txFace
	CheckLocatorMarshal(fixture.t, rxFace.Locator())
	CheckLocatorMarshal(fixture.t, txFace.Locator())

	go fixture.recvProc()
	fixture.sendProc()
	time.Sleep(800 * time.Millisecond)
	fixture.recvStop <- true
}

func (fixture *Fixture) recvProc() {
	pkts := make([]*pktmbuf.Packet, iface.MaxBurstSize)
	for {
		select {
		case <-fixture.recvStop:
			return
		default:
		}
		now := eal.TscNow()
		count, _ := fixture.rxQueueI.Pop(pkts, now)
		for _, pkt := range pkts[:count] {
			fixture.NRxInterests += fixture.recvCheck(pkt)
		}
		count, _ = fixture.rxQueueD.Pop(pkts, now)
		for _, pkt := range pkts[:count] {
			fixture.NRxData += fixture.recvCheck(pkt)
		}
		count, _ = fixture.rxQueueN.Pop(pkts, now)
		for _, pkt := range pkts[:count] {
			fixture.NRxNacks += fixture.recvCheck(pkt)
		}
	}
}

func (fixture *Fixture) recvCheck(pkt *pktmbuf.Packet) (increment int) {
	assert, _ := makeAR(fixture.t)
	faceID := iface.ID(pkt.Port())
	if fixture.rxDiscard[faceID] == nil {
		assert.Equal(fixture.rxFace.ID(), faceID)
		assert.NotZero(pkt.Timestamp())
		increment = 1
	}
	pkt.Close()
	return increment
}

func (fixture *Fixture) sendProc() {
	content := make([]byte, fixture.PayloadLen)
	for i := 0; i < fixture.TxIterations; i++ {
		pkts := make([]*ndni.Packet, 3)
		pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
		pkts[1] = ndnitestenv.MakeData("/A", content).AsPacket()
		pkts[2] = ndni.MakeNackFromInterest(ndnitestenv.MakeInterest("/A"), an.NackNoRoute).AsPacket()
		iface.TxBurst(fixture.txFace.ID(), pkts)
		time.Sleep(time.Millisecond)
	}
}

// CheckCounters checks the counters are within acceptable range.
func (fixture *Fixture) CheckCounters() {
	assert, _ := makeAR(fixture.t)

	txCnt := fixture.txFace.ReadCounters()
	assert.Equal(3*fixture.TxIterations, int(txCnt.TxFrames))
	assert.Equal(fixture.TxIterations, int(txCnt.TxInterests))
	assert.Equal(fixture.TxIterations, int(txCnt.TxData))
	assert.Equal(fixture.TxIterations, int(txCnt.TxNacks))

	rxCnt := fixture.rxFace.ReadCounters()
	assert.Equal(fixture.NRxInterests, int(rxCnt.RxInterests))
	assert.Equal(fixture.NRxData, int(rxCnt.RxData))
	assert.Equal(fixture.NRxNacks, int(rxCnt.RxNacks))
	assert.Equal(fixture.NRxInterests+fixture.NRxData+fixture.NRxNacks,
		int(rxCnt.RxFrames))

	assert.InEpsilon(fixture.TxIterations, fixture.NRxInterests, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxIterations, fixture.NRxData, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxIterations, fixture.NRxNacks, fixture.LossTolerance)
}
