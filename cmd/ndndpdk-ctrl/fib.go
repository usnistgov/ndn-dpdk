package main

import (
	"github.com/urfave/cli/v2"
)

func init() {
	defineCommand(&cli.Command{
		Name:  "list-fib",
		Usage: "List FIB entries",
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				{
					fib {
						id
						name
						nexthops {
							id
						}
						strategy {
							id
						}
					}
				}
			`, nil, "fib")
		},
	})
}
