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
	TxLoops       int     // number of TX loops
	LossTolerance float64 // permitted packet loss

	rxFace    iface.Face
	rxDiscard map[iface.ID]iface.Face
	txFace    iface.Face

	rxQueueI *iface.PktQueue
	rxQueueD *iface.PktQueue
	rxQueueN *iface.PktQueue
	rxStop   chan bool

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

// New creates a Fixture.
func New(t *testing.T, rxFace, txFace iface.Face) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.t = t

	fixture.TxLoops = 10000
	fixture.LossTolerance = 0.1

	fixture.rxFace = rxFace
	fixture.rxDiscard = make(map[iface.ID]iface.Face)
	fixture.rxStop = make(chan bool)

	fixture.txFace = txFace

	CheckLocatorMarshal(t, rxFace.Locator())
	CheckLocatorMarshal(t, txFace.Locator())
	return fixture
}

// AddRxDiscard indicates that packets received at the specified face should be dropped.
func (fixture *Fixture) AddRxDiscard(face iface.Face) {
	fixture.rxDiscard[face.ID()] = face
}

// RunTest executes the test.
func (fixture *Fixture) RunTest() {
	rxl := fixture.initRxl()
	txl := fixture.initTxl()

	go fixture.recvProc()
	fixture.sendProc()
	time.Sleep(800 * time.Millisecond)
	fixture.rxStop <- true

	txl.Close()
	rxl.Close()
	eal.Free(fixture.rxQueueI)
	eal.Free(fixture.rxQueueD)
	eal.Free(fixture.rxQueueN)
	ealthread.DefaultAllocator.Clear()
}

func (fixture *Fixture) initRxl() *iface.RxLoop {
	_, require := makeAR(fixture.t)

	rxl := iface.NewRxLoop(fixture.rxFace.NumaSocket())
	fixture.rxQueueI = fixture.preparePktQueue(rxl.InterestDemux())
	fixture.rxQueueD = fixture.preparePktQueue(rxl.DataDemux())
	fixture.rxQueueN = fixture.preparePktQueue(rxl.NackDemux())

	require.NoError(ealthread.Launch(rxl))
	time.Sleep(50 * time.Millisecond)
	for _, rxg := range fixture.rxFace.ListRxGroups() {
		require.NoError(rxl.AddRxGroup(rxg))
	}
	for _, face := range fixture.rxDiscard {
		for _, rxg := range face.ListRxGroups() {
			rxl.AddRxGroup(rxg)
		}
	}
	return rxl
}

func (fixture *Fixture) preparePktQueue(demux *iface.InputDemux) *iface.PktQueue {
	q := (*iface.PktQueue)(eal.Zmalloc("PktQueue", unsafe.Sizeof(iface.PktQueue{}), eal.NumaSocket{}))
	q.Init(iface.PktQueueConfig{}, eal.NumaSocket{})
	demux.InitFirst()
	demux.SetDest(0, q)
	return q
}

func (fixture *Fixture) initTxl() iface.TxLoop {
	_, require := makeAR(fixture.t)
	txl := iface.NewTxLoop(fixture.txFace.NumaSocket())
	require.NoError(ealthread.Launch(txl))
	txl.AddFace(fixture.txFace)
	time.Sleep(200 * time.Millisecond)
	return txl
}

func (fixture *Fixture) recvProc() {
	pkts := make([]*pktmbuf.Packet, iface.MaxBurstSize)
	for {
		select {
		case <-fixture.rxStop:
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
	for i := 0; i < fixture.TxLoops; i++ {
		pkts := make([]*ndni.Packet, 3)
		pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
		pkts[1] = ndnitestenv.MakeData("/A", content).AsPacket()
		pkts[2] = ndni.MakeNackFromInterest(ndnitestenv.MakeInterest("/A"), an.NackNoRoute).AsPacket()
		fixture.txFace.TxBurst(pkts)
		time.Sleep(time.Millisecond)
	}
}

// CheckCounters checks the counters are within acceptable range.
func (fixture *Fixture) CheckCounters() {
	assert, _ := makeAR(fixture.t)

	txCnt := fixture.txFace.ReadCounters()
	assert.Equal(3*fixture.TxLoops, int(txCnt.TxFrames))
	assert.Equal(fixture.TxLoops, int(txCnt.TxInterests))
	assert.Equal(fixture.TxLoops, int(txCnt.TxData))
	assert.Equal(fixture.TxLoops, int(txCnt.TxNacks))

	rxCnt := fixture.rxFace.ReadCounters()
	assert.Equal(fixture.NRxInterests, int(rxCnt.RxInterests))
	assert.Equal(fixture.NRxData, int(rxCnt.RxData))
	assert.Equal(fixture.NRxNacks, int(rxCnt.RxNacks))
	assert.Equal(fixture.NRxInterests+fixture.NRxData+fixture.NRxNacks,
		int(rxCnt.RxFrames))

	assert.InEpsilon(fixture.TxLoops, fixture.NRxInterests, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxData, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxNacks, fixture.LossTolerance)
}
