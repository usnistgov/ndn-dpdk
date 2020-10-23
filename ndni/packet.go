// Package ndni implements NDN layer 2 and layer 3 packet representations for internal use.
package ndni

/*
#include "../csrc/ndni/packet.h"
*/
import "C"
import (
	"encoding/binary"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

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

// PitToken retrieves the PIT token.
func (pkt *Packet) PitToken() uint64 {
	return uint64(C.Packet_GetLpL3Hdr(pkt.ptr()).pitToken)
}

// SetPitToken updates the PIT token.
func (pkt *Packet) SetPitToken(token uint64) {
	C.Packet_GetLpL3Hdr(pkt.ptr()).pitToken = C.uint64_t(token)
}

// ComputeDataImplicitDigest computes and stores implicit digest in *C.PData.
// Panics on non-Data.
func (pkt *Packet) ComputeDataImplicitDigest() []byte {
	digest := pkt.ToNPacket().Data.ComputeDigest()

	pdata := C.Packet_GetDataHdr(pkt.ptr())
	copy(cptr.AsByteSlice(&pdata.digest), digest)
	pdata.hasDigest = true

	return digest
}

// ToNPacket copies this packet into ndn.Packet.
// Panics on error.
func (pkt *Packet) ToNPacket() (npkt ndn.Packet) {
	e := tlv.Decode(pkt.Mbuf().Bytes(), &npkt)
	if e != nil {
		log.WithError(e).Panic("tlv.Decode")
	}

	lpl3 := C.Packet_GetLpL3Hdr(pkt.ptr())
	npkt.Lp.PitToken = make([]byte, 8)
	binary.BigEndian.PutUint64(npkt.Lp.PitToken, uint64(lpl3.pitToken))
	npkt.Lp.NackReason = uint8(lpl3.nackReason)
	npkt.Lp.CongMark = int(lpl3.congMark)
	if npkt.Lp.NackReason != 0 {
		return *ndn.MakeNack(npkt.Interest, npkt.Lp.NackReason).ToPacket()
	}
	return npkt
}
