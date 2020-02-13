package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Minimum dataroom of PortConfig.HeaderMp.
func SizeofTxHeader() int {
	return int(C.sizeof_struct_rte_ether_hdr) + ndn.PrependLpHeader_GetHeadroom()
}

// Port creation arguments.
type PortConfig struct {
	iface.Mempools
	RxMp      dpdk.PktmbufPool // mempool for received frames
	RxqFrames int              // RX queue capacity
	TxqPkts   int              // before-TX queue capacity
	TxqFrames int              // after-TX queue capacity
	Mtu       int              // set MTU, 0 to keep default
	Local     dpdk.EtherAddr   // local address, zero for hardware default
}

func (cfg PortConfig) check() error {
	if !cfg.Local.IsZero() && !cfg.Local.IsUnicast() {
		return errors.New("Local is not unicast")
	}
	if cfg.HeaderMp.GetDataroom() < SizeofTxHeader() {
		return errors.New("HeaderMp dataroom is too small")
	}
	return nil
}
