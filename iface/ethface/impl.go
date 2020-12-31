package ethface

import (
	"fmt"
	"io"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// RX/TX setup implementation.
type impl interface {
	fmt.Stringer
	io.Closer

	// Initialize.
	Init() error

	// Start a face.
	Start(face *ethFace) error

	// Stop a face.
	Stop(face *ethFace) error
}

type implCtor func(*Port) impl

var impls = []implCtor{newRxFlowImpl, newRxTableImpl}

// Start EthDev (called by impl).
func startDev(port *Port, nRxQueues int, promisc bool) error {
	socket := port.dev.NumaSocket()
	rxPool := ndni.PacketMempool.Get(socket)

	var cfg ethdev.Config
	cfg.AddRxQueues(nRxQueues, ethdev.RxQueueConfig{
		Capacity: port.cfg.RxQueueSize,
		Socket:   socket,
		RxPool:   rxPool,
	})
	cfg.AddTxQueues(1, ethdev.TxQueueConfig{
		Capacity: port.cfg.TxQueueSize,
		Socket:   socket,
	})

	if !port.cfg.DisableSetMTU {
		cfg.MTU = port.cfg.MTU
	}
	cfg.Promisc = promisc

	return port.dev.Start(cfg)
}
