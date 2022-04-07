package packettransport_test

import (
	"io"
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"go4.org/must"
)

var (
	makeAR = testenv.MakeAR
)

type pipePacketDataHandle struct {
	buffer []byte
	rx     net.Conn
	tx     net.Conn
}

func newPipePacketDataHandle(rx, tx net.Conn) packettransport.PacketDataHandle {
	return &pipePacketDataHandle{
		buffer: make([]byte, 4096),
		rx:     rx,
		tx:     tx,
	}
}

func (h *pipePacketDataHandle) ReadPacketData() (pkt []byte, ci gopacket.CaptureInfo, e error) {
	n, e := h.rx.Read(h.buffer)
	if e == io.ErrClosedPipe {
		return nil, ci, io.EOF
	}
	ci.CaptureLength = n
	ci.Length = n
	pkt = make([]byte, n)
	copy(pkt, h.buffer)
	return pkt, ci, e
}

func (h *pipePacketDataHandle) WritePacketData(pkt []byte) error {
	_, e := h.tx.Write(pkt)
	return e
}

func (h *pipePacketDataHandle) Close() {
	must.Close(h.rx)
	must.Close(h.tx)
}

func TestPipe(t *testing.T) {
	_, require := makeAR(t)

	var cfgA, cfgB packettransport.Config
	cfgA.Local.UnmarshalText([]byte("02:00:00:00:00:01"))
	cfgA.Remote.UnmarshalText([]byte("02:00:00:00:00:02"))
	cfgB.Local, cfgB.Remote = cfgA.Remote, cfgA.Local

	rxA, txB := net.Pipe()
	rxB, txA := net.Pipe()

	trA, e := packettransport.New(newPipePacketDataHandle(rxA, txA), cfgA)
	require.NoError(e)
	trB, e := packettransport.New(newPipePacketDataHandle(rxB, txB), cfgB)
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}
