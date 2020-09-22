package main

import (
	"github.com/urfave/cli/v2"
)

func init() {
	var name string

	defineCommand(&cli.Command{
		Category: "ndt",
		Name:     "list-ndt",
		Usage:    "List NDT entries",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "filter by `name`",
				Destination: &name,
			},
		},
		Action: func(c *cli.Context) error {
			vars := map[string]interface{}{}
			if name != "" {
				vars["name"] = name
			}

			return clientDoPrint(`
				query queryNdt($name: Name) {
					ndt(name: $name) {
						index
						value
						hits
					}
				}
			`, vars, "ndt")
		},
	})
}

func init() {
	var name string
	var value int

	defineCommand(&cli.Command{
		Category: "ndt",
		Name:     "update-ndt",
		Usage:    "Update an NDT entry",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "`name` to derive entry index",
				Destination: &name,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "value",
				Usage:       "entry `value`",
				Destination: &value,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation updateNdt($name: Name!, $value: Int!) {
					updateNdt(name: $name, value: $value) {
						index
						value
						hits
					}
				}
			`, map[string]interface{}{
				"name":  name,
				"value": value,
			}, "updateNdt")
		},
	})
}
