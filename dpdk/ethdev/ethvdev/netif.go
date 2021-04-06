package ethvdev

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/peterbourgon/mergemap"
	mathpkg "github.com/pkg/math"
	"github.com/safchain/ethtool"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/vishvananda/netlink"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	drvXDP      = "net_af_xdp_"
	drvAfPacket = "net_af_packet_"
)

// XDPProgram is the absolution path to an XDP program ELF object.
// This should be assigned by package main.
var XDPProgram string

type netIntf struct {
	*net.Interface
}

func (n netIntf) Logger() *zap.Logger {
	return logger.With(
		zap.String("netif", n.Name),
		zap.Int("ifindex", n.Index),
	)
}

func (n netIntf) PCIAddr() (a ealconfig.PCIAddress, e error) {
	device, e := filepath.EvalSymlinks(filepath.Join("/sys/class/net", n.Name, "device"))
	if e != nil {
		return ealconfig.PCIAddress{}, e
	}

	subsystem, e := filepath.EvalSymlinks(filepath.Join(device, "subsystem"))
	if e != nil || subsystem != "/sys/bus/pci" {
		return ealconfig.PCIAddress{}, e
	}

	a, e = ealconfig.ParsePCIAddress(filepath.Base(device))
	return
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
		if dev = ethdev.FromName(pciAddr.String()); dev != nil {
			return dev
		}
	}
	if dev = ethdev.FromName(drvXDP + n.Name); dev != nil {
		return dev
	}
	if dev = ethdev.FromName(drvAfPacket + n.Name); dev != nil {
		return dev
	}
	return nil
}

func (n netIntf) SetOneChannel() {
	logEntry := n.Logger()

	etht, e := ethtool.NewEthtool()
	if e != nil {
		logEntry.Error("ethtool.NewEthtool error", zap.Error(e))
		return
	}
	defer etht.Close()

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

// NetifConfig contains preferences for FromNetif.
type NetifConfig struct {
	// XDP contains preferences for net_af_xdp driver.
	XDP XDPDriverConfig `json:"xdp,omitempty"`

	// AfPacket contains preferences for net_af_packet driver.
	AfPacket AfPacketDriverConfig `json:"afPacket,omitempty"`
}

// NetifDriverConfig contains preferences for a netif-activatable driver.
type NetifDriverConfig struct {
	// Disabled prevents vdev creation with this driver.
	Disabled bool `json:"disabled,omitempty"`
	// Args overrides device arguments passed to the driver.
	Args map[string]interface{} `json:"args,omitempty"`
}

func (cfg NetifDriverConfig) makeDevImpl(drv string, netif netIntf, socket eal.NumaSocket,
	args map[string]interface{}, prepare func(args map[string]interface{}) error) (dev ethdev.EthDev, e error) {
	if cfg.Disabled {
		return nil, errors.New("driver disabled")
	}

	args = mergemap.Merge(args, cfg.Args)
	if e = prepare(args); e != nil {
		return nil, e
	}

	return New(drv+netif.Name, args, socket)
}

// XDPDriverConfig contains preferences for net_af_xdp driver.
type XDPDriverConfig struct {
	NetifDriverConfig

	// SkipSetChannels skips `ethtool --setchannels combined 1` operation.
	SkipSetChannels bool `json:"skipSetChannels,omitempty"`
}

func (cfg XDPDriverConfig) makeDev(netif netIntf, socket eal.NumaSocket) (dev ethdev.EthDev, e error) {
	args := map[string]interface{}{
		"iface":       netif.Name,
		"start_queue": 0,
		"queue_count": 1,
	}
	if XDPProgram != "" {
		args["xdp_prog"] = XDPProgram
	}

	return cfg.makeDevImpl(drvXDP, netif, socket, args, func(args map[string]interface{}) error {
		if !cfg.SkipSetChannels {
			netif.SetOneChannel()
		}
		if prog, ok := args["xdp_prog"]; ok && prog != nil {
			netif.UnloadXDP()
		}
		return nil
	})
}

// AfPacketDriverConfig contains preferences for net_af_packet driver.
type AfPacketDriverConfig struct {
	NetifDriverConfig
}

func (cfg AfPacketDriverConfig) makeDev(netif netIntf, socket eal.NumaSocket) (dev ethdev.EthDev, e error) {
	return cfg.makeDevImpl(drvAfPacket, netif, socket, map[string]interface{}{
		"iface":  netif.Name,
		"qpairs": 1,
	}, func(args map[string]interface{}) error {
		return nil
	})
}

// FromNetif finds or creates an Ethernet device.
// It can find existing PCI devices, or create a virtual device with net_af_xdp or net_af_packet driver.
func FromNetif(nif *net.Interface, cfg NetifConfig) (dev ethdev.EthDev, e error) {
	netif := netIntf{nif}
	if dev = netif.FindDev(); dev != nil {
		return dev, nil
	}

	if netif.Flags&net.FlagUp == 0 {
		return nil, errors.New("netif is not UP")
	}

	errs := []error{}
	socket := netif.NumaSocket()

	if dev, e = cfg.XDP.makeDev(netif, socket); e == nil {
		return dev, nil
	}
	errs = append(errs, fmt.Errorf("XDP %w", e))

	if dev, e = cfg.AfPacket.makeDev(netif, socket); e == nil {
		return dev, nil
	}
	errs = append(errs, fmt.Errorf("AF_PACKET %w", e))

	return nil, multierr.Combine(errs...)
}
