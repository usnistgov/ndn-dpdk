// Command ndndpdk-ctrl controls the NDN-DPDK service.
package main

import (
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/mk/version"
)

var (
	cmdout    bool
	gqlCfg    gqlclient.Config
	client    *gqlclient.Client
	interrupt = make(chan os.Signal, 1)
)

var app = &cli.App{
	Version:              version.Get().String(),
	Usage:                "Control NDN-DPDK service.",
	EnableBashCompletion: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Value:       "http://127.0.0.1:3030/",
			Destination: &gqlCfg.HTTPUri,
		},
		&cli.BoolFlag{
			Name:        "cmdout",
			Usage:       "print command line instead of running the command",
			Value:       false,
			Destination: &cmdout,
		},
	},
	Before: func(c *cli.Context) (e error) {
		signal.Notify(interrupt, syscall.SIGINT)
		if e := gqlCfg.Validate(); e != nil {
			return e
		}
		if !cmdout {
			client, e = gqlclient.New(gqlCfg)
		}
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

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
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
