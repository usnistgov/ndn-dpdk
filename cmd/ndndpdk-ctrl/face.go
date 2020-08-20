package main

import (
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

func init() {
	defineCommand(&cli.Command{
		Category: "face",
		Name:     "list-face",
		Aliases:  []string{"list-faces"},
		Usage:    "List faces.",
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
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
		Usage:    "Retrieve face information.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "Face ID.",
				Destination: &id,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "cnt",
				Usage:       "Show counters.",
				Destination: &withCounters,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				query getFace($id: ID!, $withCounters: Boolean!) {
					face: node(id: $id) {
						id
						... on Face {
							locator
							counters  @include(if: $withCounters)
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
	var loc struct {
		Scheme     string       `json:"scheme"`
		Port       string       `json:"port,omitempty"`
		Local      macaddr.Flag `json:"local"`
		Remote     macaddr.Flag `json:"remote"`
		VLAN       int          `json:"vlan,omitempty"`
		PortConfig struct {
			MTU int `json:"mtu,omitempty"`
		} `json:"portConfig"`
	}
	loc.Scheme = "ether"
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN

	defineCommand(&cli.Command{
		Category: "face",
		Name:     "create-ether-face",
		Usage:    "Create an Ethernet face.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "port",
				Usage:       "DPDK port name.",
				Destination: &loc.Port,
			},
			&cli.GenericFlag{
				Name:     "local",
				Usage:    "Local MAC address.",
				Value:    &loc.Local,
				Required: true,
			},
			&cli.GenericFlag{
				Name:  "remote",
				Usage: "Remote MAC address.",
				Value: &loc.Remote,
			},
			&cli.IntFlag{
				Name:        "vlan",
				Usage:       "VLAN identifier",
				Destination: &loc.VLAN,
			},
			&cli.IntFlag{
				Name:        "mtu",
				Usage:       "Network interface MTU",
				Destination: &loc.PortConfig.MTU,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation createFace($locator: JSON!) {
					createFace(locator: $locator) {
						id
					}
				}
			`, map[string]interface{}{
				"locator": loc,
			}, "createFace")
		},
	})
}

func init() {
	defineDeleteCommand("face", "destroy-face", "Destroy a face.")
}
