package main

import (
	"net"
	"net/netip"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/ethertransport"
)

const gqlFaceCounters = "rxFrames rxInterests rxData rxNacks txFrames txInterests txData txNacks"

func createFace(c *cli.Context, loc any) error {
	return clientDoPrint(c.Context, `
		mutation createFace($locator: JSON!) {
			createFace(locator: $locator) {
				id
			}
		}
	`, map[string]any{
		"locator": loc,
	}, "createFace")
}

func init() {
	var withCounters bool
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "list-face",
		Aliases:  []string{"list-faces"},
		Usage:    "List faces",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "cnt",
				Usage:       "show counters",
				Destination: &withCounters,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				query listFace($withCounters: Boolean!) {
					faces {
						id
						locator
						counters @include(if: $withCounters) {`+gqlFaceCounters+`}
					}
				}
			`, map[string]any{
				"withCounters": withCounters,
			}, "faces")
		},
	})
}

func init() {
	var id string
	var withCounters bool
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "get-face",
		Usage:    "Retrieve face information",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "face `ID`",
				Destination: &id,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "cnt",
				Usage:       "show counters",
				Destination: &withCounters,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				query getFace($id: ID!, $withCounters: Boolean!) {
					face: node(id: $id) {
						id
						... on Face {
							locator
							counters @include(if: $withCounters) {`+gqlFaceCounters+`}
						}
					}
				}
			`, map[string]any{
				"id":           id,
				"withCounters": withCounters,
			}, "face")
		},
	})
}

func init() {
	defineStdinJSONCommand(stdinJSONCommand{
		Category:   "face",
		Name:       "create-face",
		Usage:      "Create a face",
		SchemaName: "locator",
		ParamNoun:  "locator",
		Action: func(c *cli.Context, arg map[string]any) error {
			return createFace(c, arg)
		},
	})
}

func init() {
	var loc struct {
		Scheme      string        `json:"scheme"`
		Port        string        `json:"port,omitempty"`
		MTU         int           `json:"mtu,omitempty"`
		NRxQueues   int           `json:"nRxQueues,omitempty"`
		Local       macaddr.Flag  `json:"local"`
		Remote      macaddr.Flag  `json:"remote"`
		VLAN        int           `json:"vlan,omitempty"`
		LocalIP     *netip.Addr   `json:"localIP,omitempty"`
		RemoteIP    *netip.Addr   `json:"remoteIP,omitempty"`
		LocalUDP    *int          `json:"localUDP,omitempty"`
		RemoteUDP   *int          `json:"remoteUDP,omitempty"`
		VXLAN       int           `json:"vxlan,omitempty"`
		InnerLocal  *macaddr.Flag `json:"innerLocal,omitempty"`
		InnerRemote *macaddr.Flag `json:"innerRemote,omitempty"`
	}
	loc.Remote.HardwareAddr = ethertransport.MulticastAddressNDN

	define := func(name, usage, scheme string, remoteRequired bool, addlFlags ...cli.Flag) {
		defineCommand(&cli.Command{
			Category: "face",
			Name:     name,
			Usage:    usage,
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:        "port",
					Usage:       "DPDK `port` name",
					DefaultText: "search by local MAC address",
					Destination: &loc.Port,
				},
				&cli.IntFlag{
					Name:        "mtu",
					Usage:       "face `MTU` (excluding all headers)",
					DefaultText: "maximum",
					Destination: &loc.MTU,
				},
				&cli.IntFlag{
					Name:        "rx-queues",
					Usage:       "number of RX queues",
					DefaultText: "1",
					Destination: &loc.NRxQueues,
				},
				&cli.GenericFlag{
					Name:     "local",
					Usage:    "local MAC `address`",
					Value:    &loc.Local,
					Required: true,
				},
				&cli.GenericFlag{
					Name:     "remote",
					Usage:    "remote MAC `address`",
					Value:    &loc.Remote,
					Required: remoteRequired,
				},
				&cli.IntFlag{
					Name:        "vlan",
					Usage:       "`VLAN` identifier",
					DefaultText: "no VLAN",
					Destination: &loc.VLAN,
				},
			}, addlFlags...),
			Action: func(c *cli.Context) error {
				loc.Scheme = scheme
				return createFace(c, loc)
			},
		})
	}

	define("create-ether-face", "Create an Ethernet face", "ether", false)

	resolveUDPFlag := func(s string) (*netip.Addr, *int, error) {
		addr, e := net.ResolveUDPAddr("udp", s)
		if e != nil {
			return nil, nil, e
		}
		ip := addr.AddrPort().Addr()
		return &ip, &addr.Port, nil
	}
	define("create-udp-face", "Create a UDP face (using EthDev)", "udpe", true,
		&cli.StringFlag{
			Name:     "udp-local",
			Usage:    "local UDP `host:port`",
			Required: true,
			Action: func(c *cli.Context, s string) (e error) {
				loc.LocalIP, loc.LocalUDP, e = resolveUDPFlag(s)
				return
			},
		},
		&cli.StringFlag{
			Name:     "udp-remote",
			Usage:    "remote UDP `host:port`",
			Required: true,
			Action: func(c *cli.Context, s string) (e error) {
				loc.RemoteIP, loc.RemoteUDP, e = resolveUDPFlag(s)
				return
			},
		},
	)

	resolveIPFlag := func(s string) (*netip.Addr, error) {
		addr, e := net.ResolveIPAddr("ip", s)
		if e != nil {
			return nil, e
		}
		ip, _ := netip.AddrFromSlice(addr.IP)
		return &ip, nil
	}
	var innerLocal, innerRemote macaddr.Flag
	define("create-vxlan-face", "Create a VXLAN face", "vxlan", true,
		&cli.StringFlag{
			Name:     "ip-local",
			Usage:    "local IP `host`",
			Required: true,
			Action: func(c *cli.Context, s string) (e error) {
				loc.LocalIP, e = resolveIPFlag(s)
				return
			},
		},
		&cli.StringFlag{
			Name:     "ip-remote",
			Usage:    "remote IP `host`",
			Required: true,
			Action: func(c *cli.Context, s string) (e error) {
				loc.RemoteIP, e = resolveIPFlag(s)
				return
			},
		},
		&cli.IntFlag{
			Name:        "vxlan",
			Usage:       "`VXLAN` virtual network identifier",
			Destination: &loc.VXLAN,
			Required:    true,
		},
		&cli.GenericFlag{
			Name:     "inner-local",
			Usage:    "VXLAN inner local MAC `address`",
			Value:    &innerLocal,
			Required: true,
			Action: func(c *cli.Context, value any) error {
				loc.InnerLocal = &innerLocal
				return nil
			},
		},
		&cli.GenericFlag{
			Name:     "inner-remote",
			Usage:    "VXLAN inner remote MAC `address`",
			Value:    &innerRemote,
			Required: true,
			Action: func(c *cli.Context, value any) error {
				loc.InnerRemote = &innerRemote
				return nil
			},
		},
	)
}

func init() {
	defineDeleteCommand("face", "destroy-face", "Destroy a face", "face")
}
