// Package ethvdev facilitates virtual Ethernet devices.
package ethvdev

import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"go.uber.org/zap"
)

var logger = logging.New("ethvdev")

// New creates a virtual Ethernet device.
// The VDev will be destroyed when the EthDev is stopped and detached.
func New(name string, args map[string]interface{}, socket eal.NumaSocket) (ethdev.EthDev, error) {
	vdev, e := eal.NewVDev(name, args, socket)
	if e != nil {
		return nil, e
	}

	dev := ethdev.FromName(name)
	if dev == nil {
		logger.Panic("unexpected ethdev.FromName error",
			zap.String("name", name),
		)
	}

	ethdev.OnDetach(dev, func() { vdev.Close() })
	return dev, nil
}
