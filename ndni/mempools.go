package ndni

/*
#include "../csrc/ndni/packet.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Predefined mempool templates.
var (
	// PacketMempool is a mempool template for receiving packets.
	// This is an alias of pktmbuf.Direct.
	PacketMempool pktmbuf.Template

	// IndirectMempool is a mempool template for referencing buffers.
	// This is an alias of pktmbuf.Indirect.
	IndirectMempool pktmbuf.Template

	// HeaderMempool is a mempool template for packet headers.
	// This includes T-L portion of an L3 packet, NDNLP header, and Ethernet header.
	// It is also used for Interest guiders.
	HeaderMempool pktmbuf.Template

	// InterestMempool is a mempool template for encoding Interests.
	InterestMempool pktmbuf.Template

	// DataMempool is a mempool template for encoding Data headers.
	DataMempool pktmbuf.Template

	// PayloadMempool is a mempool template for encoding Data payload.
	PayloadMempool pktmbuf.Template
)

func init() {
	PacketMempool = pktmbuf.Direct
	PacketMempool.Update(pktmbuf.PoolConfig{
		PrivSize: C.sizeof_PacketPriv,
	})

	IndirectMempool = pktmbuf.Indirect

	const headerDataroom = pktmbuf.DefaultHeadroom + LpHeaderHeadroom
	HeaderMempool = pktmbuf.RegisterTemplate("HEADER", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: C.sizeof_PacketPriv,
		Dataroom: headerDataroom + L3TypeLengthHeadroom, // Interest TL for Interest_ModifyGuiders
	})

	InterestMempool = pktmbuf.RegisterTemplate("INTEREST", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: C.sizeof_PacketPriv,
		Dataroom: headerDataroom + InterestTemplateDataroom,
	})

	DataMempool = pktmbuf.RegisterTemplate("DATA", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: C.sizeof_PacketPriv,
		Dataroom: headerDataroom + DataGenDataroom,
	})

	PayloadMempool = pktmbuf.RegisterTemplate("PAYLOAD", pktmbuf.PoolConfig{
		Capacity: 1023,
		PrivSize: C.sizeof_PacketPriv,
		Dataroom: headerDataroom + DataGenBufLen + 9000,
	})
}

// Mempools is a set of mempools for packet modification.
type Mempools C.PacketMempools

// Assign creates mempools from templates.
//
// To use alternate templates:
//  Assign([HeaderMempool[, PacketMempool]])
func (mp *Mempools) Assign(socket eal.NumaSocket, tpl ...pktmbuf.Template) {
	if len(tpl) < 1 {
		tpl = append(tpl, HeaderMempool)
	}
	if len(tpl) < 2 {
		tpl = append(tpl, PacketMempool)
	}

	mp.packet = (*C.struct_rte_mempool)(tpl[1].Get(socket).Ptr())
	mp.indirect = (*C.struct_rte_mempool)(IndirectMempool.Get(socket).Ptr())
	mp.header = (*C.struct_rte_mempool)(tpl[0].Get(socket).Ptr())
}
