package ethvdev

import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

var activeMemifs = make(map[string]bool)

// NewMemif creates a net_memif device.
func NewMemif(loc memiftransport.Locator) (ethdev.EthDev, error) {
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memiftransport.Locator.ToVDevArgs %w", e)
	}

	key := fmt.Sprintf("%s %d", loc.SocketName, loc.ID)
	if activeMemifs[key] {
		return nil, errors.New("duplicate SocketName+ID with existing memif device")
	}

	name := "net_memif" + eal.AllocObjectID("ethface.Memif")
	dev, e := New(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethvdev.New %w", e)
	}

	activeMemifs[key] = true
	ethdev.OnDetach(dev, func() { delete(activeMemifs, key) })
	return dev, nil
}
