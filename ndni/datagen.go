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
// Name is used as name suffix.
// Panics on error.
func (gen *DataGen) Init(m *pktmbuf.Packet, args ...interface{}) {
	data := ndn.MakeData(args...)
	wire, e := tlv.EncodeValueOnly(data)
	if e != nil {
		logger.Panic("encode Data error", zap.Error(e))
	}

	var nameL, tplSize int
	d := tlv.DecodingBuffer(wire)
DecodeLoop:
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			nameL = de.Length()
			tplSize = nameL + len(de.After)
			break DecodeLoop
		}
	}

	m.SetHeadroom(0)
	if e := m.Append(wire[len(wire)-tplSize:]); e != nil {
		logger.Panic("mbuf.Append error", zap.Error(e))
	}

	c := gen.ptr()
	*c = C.DataGen{
		tpl:     (*C.struct_rte_mbuf)(m.Ptr()),
		suffixL: C.uint16_t(nameL),
	}
}

// Close discards this DataGen.
func (gen *DataGen) Close() error {
	c := gen.ptr()
	defer func() { c.tpl = nil }()
	return pktmbuf.PacketFromPtr(unsafe.Pointer(c.tpl)).Close()
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(prefix ndn.Name, mp *Mempools, fragmentPayloadSize int) *Packet {
	prefixP := NewPName(prefix)
	defer prefixP.Free()

	pktC := C.DataGen_Encode(gen.ptr(), prefixP.lname(),
		(*C.PacketMempools)(unsafe.Pointer(mp)),
		C.PacketTxAlign{
			linearize:           fragmentPayloadSize > 0,
			fragmentPayloadSize: C.uint16_t(fragmentPayloadSize),
		})
	return PacketFromPtr(unsafe.Pointer(pktC))
}

// DataGenConfig is a JSON serializable object that can construct DataGen.
type DataGenConfig struct {
	Suffix          ndn.Name                `json:"suffix,omitempty"`
	FreshnessPeriod nnduration.Milliseconds `json:"freshnessPeriod"`
	PayloadLen      int                     `json:"payloadLen"`
}

// Apply initializes InterestTemplate.
func (cfg DataGenConfig) Apply(gen *DataGen, m *pktmbuf.Packet) {
	content := make([]byte, cfg.PayloadLen)
	gen.Init(m, cfg.Suffix, cfg.FreshnessPeriod.Duration(), content)
}
