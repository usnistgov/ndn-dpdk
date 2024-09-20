package ndni

/*
#include "../csrc/ndni/data.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"go.uber.org/zap"
)

// DataGen is a Data encoder optimized for traffic generator.
type DataGen C.DataGen

// DataGenFromPtr converts *C.DataGen pointer to DataGen.
func DataGenFromPtr(ptr unsafe.Pointer) *DataGen {
	return (*DataGen)(ptr)
}

// Ptr returns *C.DataGen pointer.
func (gen *DataGen) Ptr() unsafe.Pointer {
	return unsafe.Pointer(gen)
}

func (gen *DataGen) ptr() *C.DataGen {
	return (*C.DataGen)(gen)
}

// Init initializes a DataGen.
// m is a pktmbuf with at least DataGenBufLen + len(content) buffer size; it can be allocated from PayloadMempool.
// Arguments should be acceptable to ndn.MakeData.
// Name (used as name suffix), MetaInfo, and Content are saved; other fields are skipped.
// Panics on error.
func (gen *DataGen) Init(m *pktmbuf.Packet, args ...any) {
	data := ndn.MakeData(args...)
	wire, e := tlv.EncodeValueOnly(data)
	if e != nil {
		logger.Panic("encode Data error", zap.Error(e))
	}

	m.SetHeadroom(0)
	if e := m.Append(wire); e != nil {
		logger.Panic("insufficient dataroom", zap.Error(e))
	}
	bufBegin := unsafe.Pointer(unsafe.SliceData(m.SegmentBytes()[0]))
	bufEnd := unsafe.Add(bufBegin, len(wire))
	*gen = DataGen{
		tpl:  (*C.struct_rte_mbuf)(m.Ptr()),
		meta: unsafe.SliceData(C.DataEnc_NoMetaInfo[:]),
		contentIov: [1]C.struct_iovec{{
			iov_base: bufEnd,
		}},
	}

	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			gen.suffix = C.LName{
				value:  (*C.uint8_t)(unsafe.Add(bufEnd, -len(de.After)-de.Length())),
				length: C.uint16_t(de.Length()),
			}
		case an.TtMetaInfo:
			gen.meta = (*C.uint8_t)(unsafe.Add(bufEnd, -len(de.WireAfter())))
		case an.TtContent:
			gen.contentIov[0] = C.struct_iovec{
				iov_base: unsafe.Add(bufEnd, -len(de.After)-de.Length()),
				iov_len:  C.size_t(de.Length()),
			}
		}
	}

	C.rte_pktmbuf_adj(gen.tpl, C.uint16_t(uintptr(gen.contentIov[0].iov_base)-uintptr(bufBegin)))
	C.rte_pktmbuf_trim(gen.tpl, C.uint16_t(C.size_t(pktmbuf.PacketFromPtr(unsafe.Pointer(gen.tpl)).Len())-gen.contentIov[0].iov_len))
}

// Close discards this DataGen.
func (gen *DataGen) Close() error {
	tpl := pktmbuf.PacketFromPtr(unsafe.Pointer(gen.tpl))
	*gen = DataGen{}
	return tpl.Close()
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(prefix ndn.Name, mp *Mempools, fragmentPayloadSize int) *Packet {
	prefixP := NewPName(prefix)
	defer prefixP.Free()

	pktC := C.DataGen_Encode(gen.ptr(), prefixP.lname(),
		(*C.PacketMempools)(mp),
		C.PacketTxAlign{
			linearize:           fragmentPayloadSize > 0,
			fragmentPayloadSize: C.uint16_t(fragmentPayloadSize),
		})
	return PacketFromPtr(unsafe.Pointer(pktC))
}

// DataGenConfig is a JSON serializable object that can construct DataGen.
type DataGenConfig struct {
	Suffix          ndn.Name                `json:"suffix,omitempty"`
	FreshnessPeriod nnduration.Milliseconds `json:"freshnessPeriod,omitempty"`
	PayloadLen      int                     `json:"payloadLen,omitempty"`
}

// Apply initializes DataGen.
func (cfg DataGenConfig) Apply(gen *DataGen, m *pktmbuf.Packet) {
	content := make([]byte, cfg.PayloadLen)
	gen.Init(m, cfg.Suffix, cfg.FreshnessPeriod.Duration(), content)
}
