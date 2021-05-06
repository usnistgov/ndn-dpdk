package main

import (
	"net"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

func init() {
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "list-face",
		Aliases:  []string{"list-faces"},
		Usage:    "List faces",
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				{
					faces {
						id
						locator
					}
				}
			`, nil, "faces")
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
							counters @include(if: $withCounters) {
								rxInterests
								rxData
								rxNacks
								txInterests
								txData
								txNacks
							}
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
	var innerLocal, innerRemote macaddr.Flag
	var loc struct {
		Scheme     string `json:"scheme"`
		Port       string `json:"port,omitempty"`
		PortConfig struct {
			MTU int `json:"mtu,omitempty"`
		} `json:"portConfig,omitempty"`
		MTU         int           `json:"mtu,omitempty"`
		MaxRxQueues int           `json:"maxRxQueues,omitempty"`
		Local       macaddr.Flag  `json:"local"`
		Remote      macaddr.Flag  `json:"remote"`
		VLAN        int           `json:"vlan,omitempty"`
		LocalIP     net.IP        `json:"localIP,omitempty"`
		RemoteIP    net.IP        `json:"remoteIP,omitempty"`
		LocalUDP    int           `json:"localUDP,omitempty"`
		RemoteUDP   int           `json:"remoteUDP,omitempty"`
		VXLAN       int           `json:"vxlan,omitempty"`
		InnerLocal  *macaddr.Flag `json:"innerLocal,omitempty"`
		InnerRemote *macaddr.Flag `json:"innerRemote,omitempty"`
	}
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN
	var localUDP, remoteUDP string

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "port",
			Usage:       "DPDK `port` name",
			DefaultText: "search by local MAC address",
			Destination: &loc.Port,
		},
		&cli.IntFlag{
			Name:        "port-mtu",
			Usage:       "port `MTU` (excluding Ethernet headers)",
			DefaultText: "hardware default",
			Destination: &loc.PortConfig.MTU,
		},
		&cli.IntFlag{
			Name:        "mtu",
			Usage:       "face `MTU` (excluding all headers)",
			DefaultText: "maximum",
			Destination: &loc.MTU,
		},
		&cli.IntFlag{
			Name:        "max-rxq",
			Usage:       "maximum number of RX queues",
			DefaultText: "1",
			Destination: &loc.MaxRxQueues,
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
		&cli.StringFlag{
			Name:        "udp-local",
			Usage:       "local UDP `host:port`",
			Destination: &localUDP,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "udp-remote",
			Usage:       "remote UDP `host:port`",
			Destination: &remoteUDP,
			Required:    true,
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
	}
	flagsUpTo := func(lastFlagName string) []cli.Flag {
		for i, fl := range flags {
			if fl.Names()[0] == lastFlagName {
				return flags[:i+1]
			}
		}
		panic(lastFlagName)
	}

	resolveUDP := func(c *cli.Context) error {
		local, e := net.ResolveUDPAddr("udp", localUDP)
		if e != nil {
			return e
		}
		loc.LocalIP, loc.LocalUDP = local.IP, local.Port

		remote, e := net.ResolveUDPAddr("udp", remoteUDP)
		if e != nil {
			return e
		}
		loc.RemoteIP, loc.RemoteUDP = remote.IP, remote.Port

		return nil
	}

	makeAction := func(scheme string) cli.ActionFunc {
		return func(c *cli.Context) error {
			loc.Scheme = scheme
			return clientDoPrint(c.Context, `
				mutation createFace($locator: JSON!) {
					createFace(locator: $locator) {
						id
					}
				}
		`, map[string]interface{}{
				"locator": loc,
			}, "createFace")
		}
	}

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-ether-face",
		Usage:    "Create an Ethernet face",
		Flags:    flagsUpTo("vlan"),
		Action:   makeAction("ether"),
	})

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-udp-face",
		Usage:    "Create a UDP face (using EthDev)",
		Flags:    flagsUpTo("udp-remote"),
		Before:   resolveUDP,
		Action:   makeAction("udpe"),
	})

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-vxlan-face",
		Usage:    "Create a VXLAN face",
		Flags:    flagsUpTo("inner-remote"),
		Before: func(c *cli.Context) error {
			loc.InnerLocal = &innerLocal
			loc.InnerRemote = &innerRemote
			return resolveUDP(c)
		},
		Action: makeAction("vxlan"),
	})
}

func init() {
	defineDeleteCommand("face", "destroy-face", "Destroy a face", "face")
}

func init() {
	var withDevInfo, withStats, withFaces bool
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "list-ethdev",
		Aliases:  []string{"list-ethdevs"},
		Usage:    "List Ethernet devices",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "devinfo",
				Usage:       "show DPDK device information",
				Destination: &withDevInfo,
			},
			&cli.BoolFlag{
				Name:        "stats",
				Usage:       "show hardware statistics",
				Destination: &withStats,
			},
			&cli.BoolFlag{
				Name:        "faces",
				Usage:       "show face list",
				Destination: &withFaces,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				query getEthDev(
					$withDevInfo: Boolean!
					$withStats: Boolean!
					$withFaces: Boolean!
				) {
					ethDevs {
						id
						name
						numaSocket
						macAddr
						mtu
						isDown
						devInfo @include(if: $withDevInfo)
						stats @include(if: $withStats)
						implName
						faces @include(if: $withFaces) {
							id
							locator
						}
					}
				}
			`, map[string]interface{}{
				"withDevInfo": withDevInfo,
				"withStats":   withStats,
				"withFaces":   withFaces,
			}, "ethDevs")
		},
	})
}

func init() {
	var id string

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "reset-eth-stats",
		Usage:    "Reset hardware statistics of an Ethernet device",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "device `ID`",
				Destination: &id,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				mutation resetEthStats($id: ID!) {
					resetEthStats(id: $id) {
						id
						name
					}
				}
			`, map[string]interface{}{
				"id": id,
			}, "resetEthStats")
		},
	})
}
