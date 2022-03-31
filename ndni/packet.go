// Package ndni implements NDN layer 2 and layer 3 packet representations for internal use.
package ndni

/*
#include "../csrc/ndni/packet.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"go.uber.org/zap"
)

var logger = logging.New("ndni")

// Packet represents a NDN network layer packet with parsed LP and Interest/Data headers.
type Packet C.Packet

// PacketFromPtr converts *C.Packet or *C.struct_rte_mbuf pointer to Packet.
func PacketFromPtr(ptr unsafe.Pointer) (pkt *Packet) {
	if ptr == nil {
		return nil
	}
	return (*Packet)(C.Packet_FromMbuf((*C.struct_rte_mbuf)(ptr)))
}

// Ptr returns *C.Packet or *C.struct_rte_mbuf pointer.
func (pkt *Packet) Ptr() unsafe.Pointer {
	return unsafe.Pointer(pkt)
}

func (pkt *Packet) ptr() *C.Packet {
	return (*C.Packet)(pkt)
}

// Mbuf converts to pktmbuf.Packet.
func (pkt *Packet) Mbuf() *pktmbuf.Packet {
	return pktmbuf.PacketFromPtr(pkt.Ptr())
}

// Close discards the packet.
func (pkt *Packet) Close() error {
	return pkt.Mbuf().Close()
}

// Type returns packet type.
func (pkt *Packet) Type() PktType {
	return PktType(C.Packet_GetType(pkt.ptr()))
}

// PName returns *PName pointer.
func (pkt *Packet) PName() *PName {
	return (*PName)(C.Packet_GetName(pkt.ptr()))
}

// PitToken retrieves the PIT token.
func (pkt *Packet) PitToken() (token []byte) {
	tokenC := &C.Packet_GetLpL3Hdr(pkt.ptr()).pitToken
	token = make([]byte, int(tokenC.length))
	copy(token, cptr.AsByteSlice(tokenC.value[:]))
	return
}

// SetPitToken updates the PIT token.
func (pkt *Packet) SetPitToken(token []byte) {
	tokenC := &C.Packet_GetLpL3Hdr(pkt.ptr()).pitToken
	tokenC.length = C.uint8_t(copy(cptr.AsByteSlice(tokenC.value[:]), token))
}

// ComputeDataImplicitDigest computes and stores implicit digest in *C.PData.
// Panics on non-Data.
func (pkt *Packet) ComputeDataImplicitDigest() []byte {
	digest := pkt.ToNPacket().Data.ComputeDigest()

	pdata := C.Packet_GetDataHdr(pkt.ptr())
	copy(cptr.AsByteSlice(pdata.digest[:]), digest)
	pdata.hasDigest = true

	return digest
}

// Clone clones this packet to new mbufs, with specified alignment requirement.
// Returns nil upon allocation error.
func (pkt *Packet) Clone(mp *Mempools, align PacketTxAlign) *Packet {
	m := C.Packet_Clone(pkt.ptr(), (*C.PacketMempools)(mp), *(*C.PacketTxAlign)(unsafe.Pointer(&align)))
	return PacketFromPtr(unsafe.Pointer(m))
}

// ToNPacket copies this packet into ndn.Packet.
// Panics on error.
func (pkt *Packet) ToNPacket() (npkt ndn.Packet) {
	wire := pkt.Mbuf().Bytes()
	e := tlv.Decode(wire, &npkt)
	if e != nil {
		logger.Panic("tlv.Decode",
			zap.Error(e),
			zap.Binary("wire", wire),
		)
	}

	lpl3 := C.Packet_GetLpL3Hdr(pkt.ptr())
	npkt.Lp.PitToken = pkt.PitToken()
	npkt.Lp.NackReason = uint8(lpl3.nackReason)
	npkt.Lp.CongMark = uint8(lpl3.congMark)
	if npkt.Lp.NackReason != 0 {
		return *ndn.MakeNack(npkt.Interest, npkt.Lp.NackReason).ToPacket()
	}
	return npkt
}
