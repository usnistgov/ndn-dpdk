package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func defineActivateCommand(commandName, noun, argName string) {
	defineCommand(&cli.Command{
		Category: "activate",
		Name:     commandName,
		Usage:    "Activate ndndpdk-svc as " + noun + " (pass config via stdin)",
		Action: func(c *cli.Context) error {
			arg := make(map[string]interface{})
			decoder := json.NewDecoder(os.Stdin)
			if e := decoder.Decode(&arg); e != nil {
				return e
			}

			return clientDoPrint(fmt.Sprintf(`
				mutation activate($arg: JSON!) {
					activate(%s: $arg)
				}
			`, argName), map[string]interface{}{
				"arg": arg,
			}, "activate")
		},
	})
}

func init() {
	defineActivateCommand("activate-forwarder", "forwarder", "forwarder")
	defineActivateCommand("activate-trafficgen", "traffic generator", "trafficgen")

	defineCommand(&cli.Command{
		Category: "activate",
		Name:     "shutdown",
		Usage:    "Shutdown NDN-DPDK service",
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation shutdown {
					shutdown
				}
			`, nil, "shutdown")
		},
	})
}
