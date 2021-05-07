// Command ndndpdk-ctrl controls the NDN-DPDK service.
package main

import (
	"log"
	"os"
	"sort"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/mk/version"
)

var (
	gqlserver string
	cmdout    bool
	client    *gqlclient.Client
)

var app = &cli.App{
	Version: version.Get().String(),
	Usage:   "Control NDN-DPDK service.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Value:       "http://127.0.0.1:3030/",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Destination: &gqlserver,
		},
		&cli.BoolFlag{
			Name:        "cmdout",
			Value:       false,
			Usage:       "print command line instead of executing",
			Destination: &cmdout,
		},
	},
	Before: func(c *cli.Context) (e error) {
		if cmdout {
			return nil
		}
		client, e = gqlclient.New(gqlclient.Config{HTTPUri: gqlserver})
		return e
	},
	After: func(c *cli.Context) (e error) {
		if client != nil {
			e = client.Close()
			client = nil
		}
		return e
	},
}

func main() {
	sort.Sort(cli.CommandsByName(app.Commands))
	e := app.Run(os.Args)
	if e != nil {
		log.Fatal(e)
	}
}

func init() {
	defineCommand(&cli.Command{
		Name:  "show-version",
		Usage: "Show daemon version",
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				query version {
					version {
						version
						commit
						date
						dirty
					}
				}
			`, nil, "version")
		},
	})
}
