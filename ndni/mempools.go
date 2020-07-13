package ndni

/*
#include "../csrc/ndni/packet.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Predefined mempool templates.
var (
	// PacketMempool is a mempool template for receiving packets.
	// It is an alias of pktmbuf.Direct.
	PacketMempool pktmbuf.Template

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
		PrivSize: int(C.sizeof_PacketPriv),
	})

	headerDataroom := pktmbuf.DefaultHeadroom + LpHeaderEstimatedHeadroom
	HeaderMempool = pktmbuf.RegisterTemplate("HEADER", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: int(C.sizeof_PacketPriv),
		Dataroom: headerDataroom,
	})

	InterestMempool = pktmbuf.RegisterTemplate("INTEREST", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: int(C.sizeof_PacketPriv),
		Dataroom: headerDataroom + InterestTemplateDataroom,
	})

	DataMempool = pktmbuf.RegisterTemplate("DATA", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: int(C.sizeof_PacketPriv),
		Dataroom: headerDataroom + DataGenDataroom,
	})

	PayloadMempool = pktmbuf.RegisterTemplate("PAYLOAD", pktmbuf.PoolConfig{
		Capacity: 1023,
		PrivSize: int(C.sizeof_PacketPriv),
		Dataroom: headerDataroom + DataGenBufLen + 9000,
	})
}
