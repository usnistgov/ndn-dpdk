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

func pciAddrOf(netif *net.Interface) (a ealconfig.PCIAddress, e error) {
	device, e := filepath.EvalSymlinks(filepath.Join("/sys/class/net", netif.Name, "device"))
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

func numaSocketOf(netif *net.Interface) (socket eal.NumaSocket) {
	body, e := os.ReadFile(filepath.Join("/dev/class/net", netif.Name, "device/numa_node"))
	if e != nil {
		return eal.NumaSocket{}
	}

	i, e := strconv.ParseInt(string(body), 10, 8)
	if e != nil {
		return eal.NumaSocket{}
	}
	return eal.NumaSocketFromID(int(i))
}

func findNetifDev(netif *net.Interface) (dev ethdev.EthDev) {
	if pciAddr, e := pciAddrOf(netif); e == nil {
		if dev = ethdev.FromName(pciAddr.String()); dev != nil {
			return dev
		}
	}
	if dev = ethdev.FromName(drvXDP + netif.Name); dev != nil {
		return dev
	}
	if dev = ethdev.FromName(drvAfPacket + netif.Name); dev != nil {
		return dev
	}
	return nil
}

func unloadXDP(netif *net.Interface) {
	logEntry := logger.With(
		zap.String("netif", netif.Name),
		zap.Int("ifindex", netif.Index),
	)

	link, e := netlink.LinkByIndex(netif.Index)
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
	// DisableXDP disallows net_af_xdp driver.
	DisableXDP bool `json:"disableXdp,omitempty"`

	// XDPDevArgs overrides device arguments for net_af_xdp driver.
	XDPDevArgs map[string]interface{} `json:"xdpOptions,omitempty"`

	// DisableAfPacket disallows net_af_packet driver.
	DisableAfPacket bool `json:"disableAfPacket,omitempty"`

	// AfPacketDevArgs overrides device arguments for net_af_packet driver.
	AfPacketDevArgs map[string]interface{} `json:"afPacketOptions,omitempty"`
}

func (cfg NetifConfig) makeXDP(netif *net.Interface, socket eal.NumaSocket) (dev ethdev.EthDev, e error) {
	if cfg.DisableXDP {
		return nil, errors.New("driver disabled")
	}

	args := map[string]interface{}{
		"iface":       netif.Name,
		"start_queue": 0,
		"queue_count": 1,
	}
	if XDPProgram != "" {
		args["xdp_prog"] = XDPProgram
	}
	args = mergemap.Merge(args, cfg.XDPDevArgs)

	if prog, ok := args["xdp_prog"]; ok && prog != nil {
		unloadXDP(netif)
	}
	return New(drvXDP+netif.Name, args, socket)
}

func (cfg NetifConfig) makeAfPacket(netif *net.Interface, socket eal.NumaSocket) (dev ethdev.EthDev, e error) {
	if cfg.DisableAfPacket {
		return nil, errors.New("driver disabled")
	}

	args := map[string]interface{}{
		"iface":  netif.Name,
		"qpairs": 1,
	}
	args = mergemap.Merge(args, cfg.XDPDevArgs)
	return New(drvAfPacket+netif.Name, mergemap.Merge(args, cfg.XDPDevArgs), socket)
}

// FromNetif finds or creates an Ethernet device.
// It can find existing PCI devices, or create a virtual device with net_af_xdp or net_af_packet driver.
func FromNetif(netif *net.Interface, cfg NetifConfig) (dev ethdev.EthDev, e error) {
	if dev = findNetifDev(netif); dev != nil {
		return dev, nil
	}

	if netif.Flags&net.FlagUp == 0 {
		return nil, errors.New("netif is not UP")
	}

	errs := []error{}
	socket := numaSocketOf(netif)

	dev, e = cfg.makeXDP(netif, socket)
	if dev != nil {
		return dev, nil
	}
	errs = append(errs, fmt.Errorf("XDP %w", e))

	dev, e = cfg.makeAfPacket(netif, socket)
	if dev != nil {
		return dev, nil
	}
	errs = append(errs, fmt.Errorf("AF_PACKET %w", e))

	return nil, multierr.Combine(errs...)
}
