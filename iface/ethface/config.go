package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"

	"ndn-dpdk/dpdk/ethdev"
)

// Port creation arguments.
type PortConfig struct {
	RxqFrames int              // RX queue capacity
	TxqPkts   int              // before-TX queue capacity
	TxqFrames int              // after-TX queue capacity
	Mtu       int              // set MTU, 0 to keep default
	Local     ethdev.EtherAddr // local address, zero for hardware default
}

func (cfg PortConfig) check() error {
	if !cfg.Local.IsZero() && !cfg.Local.IsUnicast() {
		return errors.New("Local is not unicast")
	}
	return nil
}
