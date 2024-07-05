package ethdev

import (
	"errors"
	"fmt"
	"net"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// NewTap creates a net_tap device.
func NewTap(ifname string, local net.HardwareAddr) (EthDev, error) {
	if !macaddr.IsUnicast(local) {
		return nil, errors.New("not unicast MAC address")
	}
	args := map[string]any{
		"iface": ifname,
		"mac":   local.String(),
	}

	name := DriverTAP + eal.AllocObjectID("ethdev.Tap")
	dev, e := NewVDev(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethdev.NewVDev(%s) %w", name, e)
	}

	return dev, nil
}
