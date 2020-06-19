package ndni

/*
#include "../csrc/ndn/packet.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Predefined mempool templates.
var (
	// PacketMempool is a mempool template for packets (mainly receiving).
	// It is an alias of pktmbuf.Direct.
	PacketMempool pktmbuf.Template

	// NameMempool is a mempool template for name linearize.
	NameMempool pktmbuf.Template

	// HeaderMempool is a mempool template for packet headers.
	// This includes T-L portion of an L3 packet, NDNLP header, and Ethernet header.
	HeaderMempool pktmbuf.Template

	// GuiderMempool is a mempool template for modifying Interest guiders.
	GuiderMempool pktmbuf.Template
)

func init() {
	PacketMempool = pktmbuf.Direct
	PacketMempool.Update(pktmbuf.PoolConfig{
		PrivSize: int(C.sizeof_PacketPriv),
	})

	NameMempool = pktmbuf.RegisterTemplate("NAME", pktmbuf.PoolConfig{
		Capacity: 65535,
		Dataroom: NameMaxLength,
	})

	HeaderMempool = pktmbuf.RegisterTemplate("HEADER", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: int(C.sizeof_PacketPriv),
		Dataroom: pktmbuf.DefaultHeadroom + LpHeaderEstimatedHeadroom,
	})

	GuiderMempool = pktmbuf.RegisterTemplate("GUIDER", pktmbuf.PoolConfig{
		Capacity: 65535,
		Dataroom: pktmbuf.DefaultHeadroom,
	})
}
