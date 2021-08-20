package ethface

import (
	"fmt"
	"io"
	"reflect"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// RX/TX setup implementation.
type impl interface {
	fmt.Stringer
	io.Closer

	// Initialize.
	Init(port *Port) error

	// Start a face.
	Start(face *ethFace) error

	// Stop a face.
	Stop(face *ethFace) error
}

var impls = []reflect.Type{
	reflect.TypeOf(rxMemifImpl{}),
	reflect.TypeOf(rxFlowImpl{}),
	reflect.TypeOf(rxTableImpl{}),
}

// Start EthDev (called by impl).
func startDev(port *Port, nRxQueues int, promisc bool) error {
	socket := port.dev.NumaSocket()
	rxPool := port.rxBouncePool
	if rxPool == nil {
		rxPool = ndni.PacketMempool.Get(socket)
	}

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
