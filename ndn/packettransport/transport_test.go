package packettransport_test

import (
	"io"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"go4.org/must"
)

var (
	makeAR = testenv.MakeAR
)

func TestAddress(t *testing.T) {
	assert, _ := makeAR(t)
	assert.True(macaddr.IsMulticast(packettransport.MulticastAddressNDN))
}

type pipePacketDataHandle struct {
	buffer []byte
	rx     net.Conn
	tx     net.Conn
	closed bool
}

func newPipePacketDataHandle(rx, tx net.Conn) *pipePacketDataHandle {
	return &pipePacketDataHandle{
		buffer: make([]byte, 4096),
		rx:     rx,
		tx:     tx,
	}
}

func (h *pipePacketDataHandle) ZeroCopyReadPacketData() (pkt []byte, ci gopacket.CaptureInfo, e error) {
	n, e := h.rx.Read(h.buffer)
	if e == io.ErrClosedPipe {
		return nil, ci, io.EOF
	}
	ci.CaptureLength = n
	ci.Length = n
	return h.buffer[:n], ci, e
}

func (h *pipePacketDataHandle) WritePacketData(pkt []byte) error {
	_, e := h.tx.Write(pkt)
	return e
}

func (h *pipePacketDataHandle) Close() {
	must.Close(h.rx)
	must.Close(h.tx)
	h.closed = true
}

func TestPipe(t *testing.T) {
	assert, require := makeAR(t)

	var cfgA packettransport.Config
	cfgA.MTU = 4096
	cfgA.Local.UnmarshalText([]byte("02:00:00:00:00:01"))
	cfgA.Remote.UnmarshalText([]byte("02:00:00:00:00:02"))
	cfgB := cfgA
	cfgB.Local, cfgB.Remote = cfgA.Remote, cfgA.Local

	rxA, txB := net.Pipe()
	rxB, txA := net.Pipe()

	hA := newPipePacketDataHandle(rxA, txA)
	hB := newPipePacketDataHandle(rxB, txB)
	trA, e := packettransport.New(hA, cfgA)
	require.NoError(e)
	trB, e := packettransport.New(hB, cfgB)
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)

	assert.True(hA.closed)
	assert.True(hB.closed)
}
