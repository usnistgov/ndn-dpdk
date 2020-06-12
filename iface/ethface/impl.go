package ethface

import (
	"fmt"
	"io"

	"ndn-dpdk/dpdk/ethdev"
	"ndn-dpdk/ndn"
)

// RX/TX setup implementation.
type iImpl interface {
	fmt.Stringer
	io.Closer

	// Construct new instance.
	New(port *Port) iImpl

	// Initialize.
	Init() error

	// Start a face.
	Start(face *EthFace) error

	// Stop a face.
	Stop(face *EthFace) error
}

var impls = []iImpl{&rxFlowImpl{}, &rxTableImpl{}}

// Start EthDev (called by impl).
func startDev(port *Port, nRxQueues int, promisc bool) error {
	socket := port.dev.GetNumaSocket()
	var cfg ethdev.Config
	cfg.AddRxQueues(nRxQueues, ethdev.RxQueueConfig{
		Capacity: port.cfg.RxqFrames,
		Socket:   socket,
		RxPool:   ndn.PacketMempool.MakePool(socket),
	})
	cfg.AddTxQueues(1, ethdev.TxQueueConfig{
		Capacity: port.cfg.TxqFrames,
		Socket:   socket,
	})
	cfg.Mtu = port.cfg.Mtu
	cfg.Promisc = promisc
	return port.dev.Start(cfg)
}
