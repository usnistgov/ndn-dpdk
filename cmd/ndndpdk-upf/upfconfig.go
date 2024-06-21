package main

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/wmnsk/go-pfcp/ie"
)

type UpfConfig struct {
	SmfN4   netip.Addr
	UpfN4   netip.Addr
	upfIP   netip.Addr
	upfMAC  macaddr.Flag
	upfVLAN int
	mapN3   map[netip.Addr]macaddr.Flag
	dnIP    netip.Addr

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
			Action:   cfg.saveIPv4(&cfg.upfIP),
		},
		&cli.GenericFlag{
			Name:        "upf-mac",
			Usage:       "UPF N3 MAC `address`",
			Required:    true,
			Destination: &cfg.upfMAC,
		},
		&cli.IntFlag{
			Name:        "upf-vlan",
			Usage:       "UPF N3 `VLAN ID`",
			Destination: &cfg.upfVLAN,
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
			Action:   cfg.saveIPv4(&cfg.dnIP),
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
	if !macaddr.IsUnicast(cfg.upfMAC.HardwareAddr) {
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

func (cfg UpfConfig) MakeLocator(ulTEID uint32, ulQFI uint8, dlTEID uint32, dlQFI uint8, peer, ueIP netip.Addr) (loc map[string]any, e error) {
	loc = map[string]any{
		"scheme":        "gtp",
		"local":         cfg.upfMAC,
		"localIP":       cfg.upfIP,
		"ulTEID":        ulTEID,
		"ulQFI":         ulQFI,
		"dlTEID":        dlTEID,
		"dlQFI":         dlQFI,
		"innerLocalIP":  cfg.dnIP,
		"innerRemoteIP": ueIP,
	}
	if cfg.upfVLAN > 0 {
		loc["vlan"] = cfg.upfVLAN
	}
	if remote, ok := cfg.mapN3[peer]; ok {
		loc["remote"], loc["remoteIP"] = remote, peer
	} else {
		return nil, fmt.Errorf("unknown MAC address for peer %s", peer)
	}
	return loc, nil
}
