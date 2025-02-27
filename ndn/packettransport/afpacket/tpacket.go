package afpacket

import (
	"io"
	"reflect"

	"github.com/gopacket/gopacket/afpacket"
)

// TPacketHandle is a wrapper of afpacket.TPacket.
type TPacketHandle struct {
	*afpacket.TPacket
}

var (
	_ io.ReadWriteCloser = (*TPacketHandle)(nil)
)

// Read implements io.Reader interface.
func (h *TPacketHandle) Read(buf []byte) (n int, e error) {
	ci, e := h.ReadPacketDataTo(buf)
	return ci.CaptureLength, e
}

// Write implements io.Writer interface.
func (h *TPacketHandle) Write(pkt []byte) (n int, e error) {
	e = h.WritePacketData(pkt)
	return len(pkt), e
}

// Close implements io.Closer interface.
func (h *TPacketHandle) Close() error {
	h.TPacket.Close()
	return nil
}

// FD returns underlying file descriptor.
func (h *TPacketHandle) FD() int {
	return int(reflect.ValueOf(h.TPacket).Elem().FieldByName("fd").Int())
}

// NewTPacketHandle wraps afpacket.TPacket.
func NewTPacketHandle(tpacket *afpacket.TPacket) (h *TPacketHandle) {
	return &TPacketHandle{
		TPacket: tpacket,
	}
}
