package ethnetif

import (
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"

	mathpkg "github.com/pkg/math"
	"github.com/safchain/ethtool"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

var logger = logging.New("ethnetif")

var etht *ethtool.Ethtool

type netIntf struct {
	*net.Interface
}

func (n netIntf) Logger() *zap.Logger {
	return logger.With(
		zap.String("netif", n.Name),
		zap.Int("ifindex", n.Index),
	)
}

func (n netIntf) PCIAddr() (a pciaddr.PCIAddress, e error) {
	busInfo, e := etht.BusInfo(n.Name)
	if e != nil {
		return pciaddr.PCIAddress{}, e
	}

	return pciaddr.Parse(filepath.Base(busInfo))
}

func (n netIntf) NumaSocket() (socket eal.NumaSocket) {
	body, e := os.ReadFile(filepath.Join("/dev/class/net", n.Name, "device/numa_node"))
	if e != nil {
		return eal.NumaSocket{}
	}

	i, e := strconv.ParseInt(string(body), 10, 8)
	if e != nil {
		return eal.NumaSocket{}
	}
	return eal.NumaSocketFromID(int(i))
}

func (n netIntf) FindDev() (dev ethdev.EthDev) {
	if pciAddr, e := n.PCIAddr(); e == nil {
		if dev = ethdev.FromPCI(pciAddr); dev != nil {
			return dev
		}
	}
	if dev = ethdev.FromName(ethdev.DriverXDP + "_" + n.Name); dev != nil {
		return dev
	}
	if dev = ethdev.FromName(ethdev.DriverAfPacket + "_" + n.Name); dev != nil {
		return dev
	}
	return nil
}

func (n netIntf) SetOneChannel() {
	logEntry := n.Logger()

	channels, e := etht.GetChannels(n.Name)
	if e != nil {
		logEntry.Error("ethtool.GetChannels error", zap.Error(e))
		return
	}

	channelsUpdate := channels
	channelsUpdate.RxCount = mathpkg.MinUint32(channels.MaxRx, 1)
	channelsUpdate.CombinedCount = mathpkg.MinUint32(channels.MaxCombined, 1)

	logEntry = logEntry.With(
		zap.Uint32("old-rx", channels.RxCount),
		zap.Uint32("old-combined", channels.CombinedCount),
		zap.Uint32("new-rx", channelsUpdate.RxCount),
		zap.Uint32("new-combined", channelsUpdate.CombinedCount),
	)

	if channelsUpdate == channels {
		logEntry.Debug("no change in channels")
		return
	}

	_, e = etht.SetChannels(n.Name, channelsUpdate)
	if e != nil {
		logEntry.Error("ethtool.SetChannels error", zap.Error(e))
		return
	}

	logEntry.Debug("changed to 1 channel")
}

func (n netIntf) DisableVLANOffload() {
	logEntry := n.Logger()

	features, e := etht.Features(n.Name)
	if e != nil {
		logEntry.Error("ethtool.Features error", zap.Error(e))
		return
	}

	const rxvlanKey = "rx-vlan-hw-parse"
	rxvlan, ok := features[rxvlanKey]
	if !ok {
		logEntry.Debug("rxvlan offload not supported")
		return
	}
	if !rxvlan {
		logEntry.Debug("rxvlan offload already disabled")
		return
	}

	e = etht.Change(n.Name, map[string]bool{
		rxvlanKey: false,
	})
	if e != nil {
		logEntry.Error("ethtool.Change(rxvlan=false) error", zap.Error(e))
		return
	}

	logEntry.Debug("disabled rxvlan offload")
}

func (n netIntf) UnloadXDP() {
	logEntry := n.Logger()

	link, e := netlink.LinkByIndex(n.Index)
	if e != nil {
		logEntry.Error("netlink.LinkByIndex error", zap.Error(e))
		return
	}
	attrs := link.Attrs()

	if attrs.Xdp == nil || !attrs.Xdp.Attached {
		logEntry.Debug("netlink has no attached XDP program")
		return
	}
	logEntry = logEntry.With(zap.Uint32("old-xdp-prog", attrs.Xdp.ProgId))

	e = netlink.LinkSetXdpFd(link, math.MaxUint32)
	if e != nil {
		logEntry.Error("netlink.LinkSetXdpFd error", zap.Error(e))
		return
	}

	logEntry.Debug("unloaded previous XDP program")
}

func netIntfByName(ifname string) (netIntf, error) {
	nif, e := net.InterfaceByName(ifname)
	if e != nil {
		return netIntf{}, fmt.Errorf("net.InterfaceByName(%s): %w", ifname, e)
	}
	return netIntf{nif}, nil
}
