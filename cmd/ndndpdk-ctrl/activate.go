package main

import (
	"github.com/urfave/cli/v2"
)

func defineActivateCommand(id, noun string) {
	defineStdinJSONCommand(stdinJSONCommand{
		Category:   "activate",
		Name:       "activate-" + id,
		Usage:      "Activate ndndpdk-svc as " + noun,
		SchemaName: id,
		Action: func(c *cli.Context, arg map[string]interface{}) error {
			return clientDoPrint(c.Context, `
				mutation activate($arg: JSON!) {
					activate(`+id+`: $arg)
				}
			`, map[string]interface{}{
				"arg": arg,
			}, "activate")
		},
	})
}

func init() {
	defineActivateCommand("forwarder", "forwarder")
	defineActivateCommand("trafficgen", "traffic generator")

	defineCommand(&cli.Command{
		Category: "activate",
		Name:     "shutdown",
		Usage:    "Shutdown NDN-DPDK service",
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				mutation shutdown {
					shutdown
				}
			`, nil, "shutdown")
		},
	})
}
