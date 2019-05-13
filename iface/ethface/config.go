package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Minimum dataroom of PortConfig.HeaderMp.
func SizeofTxHeader() int {
	return int(C.sizeof_struct_ether_hdr) + ndn.PrependLpHeader_GetHeadroom()
}

// Port creation arguments.
type PortConfig struct {
	iface.Mempools
	RxMp      dpdk.PktmbufPool // mempool for received frames
	RxqFrames int              // RX queue capacity
	TxqPkts   int              // before-TX queue capacity
	TxqFrames int              // after-TX queue capacity
	Mtu       int              // set MTU, 0 to keep default
	Local     net.HardwareAddr // local address, nil for hardware default
}

func (cfg PortConfig) check() error {
	if cfg.Local != nil && classifyMac48(cfg.Local) != mac48_unicast {
		return errors.New("cfg.Local is not a MAC-48 unicast address")
	}
	if cfg.HeaderMp.GetDataroom() < SizeofTxHeader() {
		return errors.New("cfg.HeaderMp dataroom is too small")
	}
	return nil
}
