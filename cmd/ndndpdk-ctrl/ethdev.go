package main

import (
	"errors"

	"github.com/urfave/cli/v2"
)

const gqlEthDevFields = "id name numaSocket macAddr mtu isDown"

func init() {
	var withDevInfo, withStats, withFaces bool
	defineCommand(&cli.Command{
		Category: "ethdev",
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
				query listEthDev(
					$withDevInfo: Boolean!
					$withStats: Boolean!
					$withFaces: Boolean!
				) {
					ethDevs {
						`+gqlEthDevFields+`
						devInfo @include(if: $withDevInfo)
						stats @include(if: $withStats)
						faces @include(if: $withFaces) {
							id
							locator
						}
					}
				}
			`, map[string]any{
				"withDevInfo": withDevInfo,
				"withStats":   withStats,
				"withFaces":   withFaces,
			}, "ethDevs")
		},
	})
}

func init() {
	defineCommand(&cli.Command{
		Category: "ethdev",
		Name:     "create-eth-port",
		Usage:    "Create an Ethernet port",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "pci",
				Usage: "use PCI driver with `PCI address`",
			},
			&cli.StringFlag{
				Name:  "netif",
				Usage: "network `interface` name",
			},
			&cli.BoolFlag{
				Name:        "xdp",
				Usage:       "use XDP driver on netif",
				DefaultText: "use AF_PACKET driver",
			},
			&cli.UintFlag{
				Name:        "mtu",
				Usage:       "set interface `MTU`",
				DefaultText: "unchanged",
			},
			&cli.UintFlag{
				Name:        "rx-flow",
				Usage:       "enable RxFlow with specified number of `queues`",
				DefaultText: "disable RxFlow",
			},
		},
		Action: func(c *cli.Context) error {
			vars := map[string]any{
				"driver": "AF_PACKET",
			}
			switch {
			case c.IsSet("pci"):
				vars["driver"] = "PCI"
				vars["pciAddr"] = c.String("pci")
			case c.Bool("xdp"):
				vars["driver"] = "XDP"
				fallthrough
			default:
				if c.IsSet("netif") {
					vars["netif"] = c.String("netif")
				} else {
					return errors.New("either --pci or --netif must be set")
				}
			}
			if c.IsSet("mtu") {
				vars["mtu"] = c.Uint("mtu")
			}
			if c.IsSet("rx-flow") {
				vars["rxFlowQueues"] = c.Uint("rx-flow")
			}

			return clientDoPrint(c.Context, `
				mutation createEthPort(
					$driver: NetifDriverKind!
					$pciAddr: String
					$netif: String
					$mtu: Int
					$rxFlowQueues: Int
				) {
					createEthPort(
						driver: $driver
						pciAddr: $pciAddr
						netif: $netif
						mtu: $mtu
						rxFlowQueues: $rxFlowQueues
					) {`+gqlEthDevFields+`}
				}
			`, vars, "createEthPort")
		},
	})
}

func init() {
	var id string
	defineCommand(&cli.Command{
		Category: "ethdev",
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
			`, map[string]any{
				"id": id,
			}, "resetEthStats")
		},
	})
}
