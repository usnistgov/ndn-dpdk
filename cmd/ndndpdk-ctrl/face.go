package main

import (
	"net"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"inet.af/netaddr"
)

const gqlFaceCounters = "rxFrames rxInterests rxData rxNacks txFrames txInterests txData txNacks"

const gqlCreateFace = `
	mutation createFace($locator: JSON!) {
		createFace(locator: $locator) {
			id
		}
	}
`

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
			`, map[string]interface{}{
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
			`, map[string]interface{}{
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
		Action: func(c *cli.Context, arg map[string]interface{}) error {
			return clientDoPrint(c.Context, gqlCreateFace, map[string]interface{}{
				"locator": arg,
			}, "createFace")
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
		LocalIP     *netaddr.IP   `json:"localIP,omitempty"`
		RemoteIP    *netaddr.IP   `json:"remoteIP,omitempty"`
		LocalUDP    *int          `json:"localUDP,omitempty"`
		RemoteUDP   *int          `json:"remoteUDP,omitempty"`
		VXLAN       int           `json:"vxlan,omitempty"`
		InnerLocal  *macaddr.Flag `json:"innerLocal,omitempty"`
		InnerRemote *macaddr.Flag `json:"innerRemote,omitempty"`
	}
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN

	ethFlags := []cli.Flag{
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
			Name:  "remote",
			Usage: "remote MAC `address`",
			Value: &loc.Remote,
		},
		&cli.IntFlag{
			Name:        "vlan",
			Usage:       "`VLAN` identifier",
			DefaultText: "no VLAN",
			Destination: &loc.VLAN,
		},
	}

	makeAction := func(scheme string) cli.ActionFunc {
		return func(c *cli.Context) error {
			loc.Scheme = scheme
			return clientDoPrint(c.Context, gqlCreateFace, map[string]interface{}{
				"locator": loc,
			}, "createFace")
		}
	}

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-ether-face",
		Usage:    "Create an Ethernet face",
		Flags:    ethFlags,
		Action:   makeAction("ether"),
	})

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-udp-face",
		Usage:    "Create a UDP face (using EthDev)",
		Flags: append(append([]cli.Flag{}, ethFlags...),
			&cli.StringFlag{
				Name:     "udp-local",
				Usage:    "local UDP `host:port`",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "udp-remote",
				Usage:    "remote UDP `host:port`",
				Required: true,
			},
		),
		Before: func(c *cli.Context) error {
			local, e := net.ResolveUDPAddr("udp", c.String("udp-local"))
			if e != nil {
				return e
			}
			localIP, _ := netaddr.FromStdIP(local.IP)
			loc.LocalIP, loc.LocalUDP = &localIP, &local.Port

			remote, e := net.ResolveUDPAddr("udp", c.String("udp-remote"))
			if e != nil {
				return e
			}
			remoteIP, _ := netaddr.FromStdIP(remote.IP)
			loc.RemoteIP, loc.RemoteUDP = &remoteIP, &remote.Port

			return nil
		},
		Action: makeAction("udpe"),
	})

	var innerLocal, innerRemote macaddr.Flag
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-vxlan-face",
		Usage:    "Create a VXLAN face",
		Flags: append(append([]cli.Flag{}, ethFlags...),
			&cli.StringFlag{
				Name:     "ip-local",
				Usage:    "local IP `host`",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "ip-remote",
				Usage:    "remote IP `host`",
				Required: true,
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
			},
			&cli.GenericFlag{
				Name:     "inner-remote",
				Usage:    "VXLAN inner remote MAC `address`",
				Value:    &innerRemote,
				Required: true,
			},
		),
		Before: func(c *cli.Context) error {
			local, e := net.ResolveIPAddr("ip", c.String("ip-local"))
			if e != nil {
				return e
			}
			localIP, _ := netaddr.FromStdIP(local.IP)
			loc.LocalIP = &localIP

			remote, e := net.ResolveIPAddr("ip", c.String("ip-remote"))
			if e != nil {
				return e
			}
			remoteIP, _ := netaddr.FromStdIP(remote.IP)
			loc.RemoteIP = &remoteIP

			loc.InnerLocal = &innerLocal
			loc.InnerRemote = &innerRemote
			return nil
		},
		Action: makeAction("vxlan"),
	})
}

func init() {
	defineDeleteCommand("face", "destroy-face", "Destroy a face", "face")
}
