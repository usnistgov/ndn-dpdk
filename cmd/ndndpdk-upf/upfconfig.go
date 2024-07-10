package main

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/wmnsk/go-pfcp/ie"
)

type UpfLocatorFields struct {
	Scheme       string       `json:"scheme"`
	Local        macaddr.Flag `json:"local"`
	VLAN         int          `json:"vlan,omitempty"`
	LocalIP      netip.Addr   `json:"localIP"`
	InnerLocalIP netip.Addr   `json:"innerLocalIP"`
}

type UpfConfig struct {
	SmfN4 netip.Addr
	UpfN4 netip.Addr
	loc   UpfLocatorFields
	mapN3 map[netip.Addr]macaddr.Flag

	RecoveryTimestamp *ie.IE
	UpfNodeID         *ie.IE
}

func (cfg *UpfConfig) DefineFlags(flags []cli.Flag) []cli.Flag {
	return append(flags,
		&cli.StringFlag{
			Name:     "smf-n4",
			Usage:    "SMF N4 IPv4 `address`",
			Required: true,
			Action:   cfg.saveIPv4(&cfg.SmfN4),
		},
		&cli.StringFlag{
			Name:     "upf-n4",
			Usage:    "UPF N4 IPv4 `address`",
			Required: true,
			Action:   cfg.saveIPv4(&cfg.UpfN4),
		},
		&cli.StringFlag{
			Name:     "upf-n3",
			Usage:    "UPF N3 IPv4 `address`",
			Required: true,
			Action:   cfg.saveIPv4(&cfg.loc.LocalIP),
		},
		&cli.GenericFlag{
			Name:        "upf-mac",
			Usage:       "UPF N3 MAC `address`",
			Required:    true,
			Destination: &cfg.loc.Local,
		},
		&cli.IntFlag{
			Name:        "upf-vlan",
			Usage:       "UPF N3 `VLAN ID`",
			Destination: &cfg.loc.VLAN,
		},
		&cli.StringSliceFlag{
			Name:     "n3",
			Usage:    "N3 `ip=mac` tuple",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dn",
			Usage:    "Data Network NDN forwarder IPv4 `address`",
			Required: true,
			Action:   cfg.saveIPv4(&cfg.loc.InnerLocalIP),
		},
	)
}

func (UpfConfig) saveIPv4(d *netip.Addr) func(c *cli.Context, v string) error {
	return func(c *cli.Context, v string) (e error) {
		if *d, e = netip.ParseAddr(v); e != nil || !d.Is4() {
			return fmt.Errorf("'%s' is not an IPv4 address", v)
		}
		return nil
	}
}

func (cfg *UpfConfig) ProcessFlags(c *cli.Context) error {
	cfg.loc.Scheme = "gtp"
	if !macaddr.IsUnicast(cfg.loc.Local.HardwareAddr) {
		return errors.New("upf-mac is not unicast MAC address")
	}

	cfg.mapN3 = map[netip.Addr]macaddr.Flag{}
	for i, line := range c.StringSlice("n3") {
		tokens := strings.Split(line, "=")
		if len(tokens) != 2 {
			return fmt.Errorf("n3[%d] is invalid", i)
		}
		n3ip, e := netip.ParseAddr(tokens[0])
		if e != nil || !n3ip.Is4() {
			return fmt.Errorf("'%s' is not an IPv4 address", tokens[0])
		}
		var n3mac macaddr.Flag
		if e := n3mac.Set(tokens[1]); e != nil || !macaddr.IsUnicast(n3mac.HardwareAddr) {
			return fmt.Errorf("'%s' is not a unicast MAC address", tokens[1])
		}
		cfg.mapN3[n3ip] = n3mac
	}

	cfg.RecoveryTimestamp = ie.NewRecoveryTimeStamp(time.Now())
	cfg.UpfNodeID = ie.NewNodeID(cfg.UpfN4.String(), "", "")
	return nil
}

func (cfg UpfConfig) MakeLocator(sloc upf.SessionLocatorFields) (any, error) {
	loc := struct {
		upf.SessionLocatorFields
		UpfLocatorFields
		Remote macaddr.Flag `json:"remote"`
	}{
		SessionLocatorFields: sloc,
		UpfLocatorFields:     cfg.loc,
	}

	if remote, ok := cfg.mapN3[loc.RemoteIP]; ok {
		loc.Remote = remote
	} else {
		return nil, fmt.Errorf("unknown MAC address for peer %s", loc.RemoteIP)
	}
	return loc, nil
}
