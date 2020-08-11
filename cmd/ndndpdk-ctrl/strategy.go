package main

import (
	"github.com/urfave/cli/v2"
)

func init() {
	var withFib bool

	defineCommand(&cli.Command{
		Name:    "list-strategy",
		Aliases: []string{"list-strategies"},
		Usage:   "List strategies",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "fib",
				Usage:       "Show FIB entries.",
				Destination: &withFib,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				query listStrategy($withFib: Boolean!) {
					strategies {
						id
						name
						fibEntries @include(if: $withFib) {
							id
							name
						}
					}
				}
			`, map[string]interface{}{
				"withFib": withFib,
			}, "strategies")
		},
	})
}
