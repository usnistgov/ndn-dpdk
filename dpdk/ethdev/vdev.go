package ethdev

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

// NewVDev creates a virtual Ethernet device.
// The VDev will be destroyed when the EthDev is stopped and detached.
func NewVDev(name string, args map[string]any, socket eal.NumaSocket) (EthDev, error) {
	vdev, e := eal.NewVDev(name, args, socket)
	if e != nil {
		return nil, e
	}

	dev := FromName(name)
	if dev == nil {
		logger.Panic("unexpected ethdev.FromName error",
			zap.String("name", name),
		)
	}

	OnClose(dev, func() { vdev.Close() })
	return dev, nil
}
