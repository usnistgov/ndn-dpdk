package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func init() {
	var name string
	var fetchOptions segmented.FetchOptions
	defineCommand(&cli.Command{
		Name:  "get",
		Usage: "Retrieve segmented object.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "retx-limit",
				Usage:       "retransmission limit",
				Destination: &fetchOptions.RetxLimit,
				Value:       15,
			},
			&cli.IntFlag{
				Name:        "max-cwnd",
				Usage:       "maximum congestion window",
				Destination: &fetchOptions.MaxCwnd,
				Value:       24,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				<-interrupt
				cancel()
			}()
			return segmented.Fetch(ndn.ParseName(name), fetchOptions).Pipe(ctx, os.Stdout)
		},
	})
}
