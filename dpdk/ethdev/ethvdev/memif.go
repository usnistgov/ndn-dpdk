package ethvdev

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

var memifCoexist = make(memiftransport.CoexistMap)

// NewMemif creates a net_memif device.
func NewMemif(loc memiftransport.Locator) (ethdev.EthDev, error) {
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memiftransport.Locator.ToVDevArgs %w", e)
	}

	if e := memifCoexist.Check(loc); e != nil {
		return nil, e
	}

	name := "net_memif" + eal.AllocObjectID("ethvdev.Memif")
	dev, e := New(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("ethvdev.New %w", e)
	}

	memifCoexist.Add(loc)
	ethdev.OnDetach(dev, func() { memifCoexist.Remove(loc) })
	return dev, nil
}
