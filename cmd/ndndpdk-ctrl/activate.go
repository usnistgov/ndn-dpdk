package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/kballard/go-shellquote"
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
	defineActivateCommand("fileserver", "file server")
}

func init() {
	restart := false
	defineCommand(&cli.Command{
		Category: "activate",
		Name:     "shutdown",
		Usage:    "Shutdown NDN-DPDK service",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "restart",
				Usage:       "restart after shutdown",
				Destination: &restart,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				mutation shutdown($restart: Boolean) {
					shutdown(restart: $restart)
				}
			`, map[string]interface{}{
				"restart": restart,
			}, "shutdown")
		},
	})
}

func init() {
	run := func(name string, arg ...string) error {
		if cmdout {
			fmt.Println("sudo", name, shellquote.Join(arg...))
			return nil
		}

		cmd := exec.Command(name, arg...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}

	unitName := ""
	logsFollow := false
	cmd := &cli.Command{
		Category: "activate",
		Name:     "systemd",
		Usage:    "Control NDN-DPDK systemd service",
		Before: func(c *cli.Context) error {
			hostport, e := gqlCfg.Listen()
			if e != nil {
				return e
			}
			unitName = "ndndpdk-svc@" + unit.UnitNameEscape(hostport) + ".service"
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:    "start",
				Aliases: []string{"restart"},
				Usage:   "Start or restart the service (requires sudo)",
				Action: func(c *cli.Context) error {
					return run("systemctl", "restart", unitName)
				},
			},
			{
				Name:  "stop",
				Usage: "Stop the service (requires sudo)",
				Action: func(c *cli.Context) error {
					return run("systemctl", "stop", unitName)
				},
			},
			{
				Name:  "logs",
				Usage: "View service logs",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "f",
						Usage:       "follow new log entries",
						Destination: &logsFollow,
					},
				},
				Action: func(c *cli.Context) error {
					if logsFollow {
						return run("journalctl", "-ocat", "-f", "-u", unitName)
					}
					return run("journalctl", "-ocat", "-u", unitName)
				},
			},
		},
	}

	sort.Sort(cli.CommandsByName(cmd.Subcommands))
	defineCommand(cmd)
}
