// +build linux,cgo

package main

import (
	"log"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
)

func init() {
	var cfg afpacket.Config
	var netif string
	var respond bool
	defineCommand(&cli.Command{
		Name:  "dump",
		Usage: "Capture traffic on an AF_PACKET face.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "netif",
				Usage:       "Network `interface` name.",
				Destination: &netif,
				Required:    true,
			},
			&cli.GenericFlag{
				Name:  "local",
				Usage: "Local MAC address.",
				Value: &cfg.Local,
			},
			&cli.GenericFlag{
				Name:  "remote",
				Usage: "Remote MAC address.",
				Value: &cfg.Remote,
			},
			&cli.BoolFlag{
				Name:        "respond",
				Usage:       "Respond every Interest with Data.",
				Destination: &respond,
			},
			&cli.IntFlag{
				Name:        "rxq",
				Usage:       "RX queue size.",
				Value:       l3.DefaultTransportRxQueueSize,
				Destination: &cfg.RxQueueSize,
			},
			&cli.IntFlag{
				Name:        "txq",
				Usage:       "TX queue size.",
				Value:       l3.DefaultTransportTxQueueSize,
				Destination: &cfg.TxQueueSize,
			},
		},
		Action: func(c *cli.Context) error {
			tr, e := afpacket.New(netif, cfg)
			if e != nil {
				return e
			}

			f, e := l3.NewFace(tr)
			if e != nil {
				return e
			}
			defer close(f.Tx())

			for {
				select {
				case <-interrupt:
					return nil
				case packet := <-f.Rx():
					log.Println(packet)
					if respond && packet.Interest != nil {
						data := ndn.MakeData(packet.Interest)
						select {
						case f.Tx() <- data:
						default:
						}
					}
				}
			}
		},
	})
}
