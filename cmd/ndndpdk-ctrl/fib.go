package main

import (
	"github.com/urfave/cli/v2"
)

func init() {
	defineCommand(&cli.Command{
		Category: "fib",
		Name:     "list-fib",
		Usage:    "List FIB entries",
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

func init() {
	var name string
	var nexthops cli.StringSlice
	var strategy string

	defineCommand(&cli.Command{
		Category: "fib",
		Name:     "insert-fib",
		Usage:    "Insert or replace a FIB entry",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.StringSliceFlag{
				Name:        "nexthop",
				Aliases:     []string{"nh"},
				Usage:       "FIB nexthop face `ID` (repeatable)",
				Destination: &nexthops,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "strategy",
				Usage:       "forwarding strategy `ID`",
				Destination: &strategy,
			},
		},
		Action: func(c *cli.Context) error {
			vars := map[string]interface{}{
				"name":     name,
				"nexthops": nexthops.Value(),
			}
			if strategy != "" {
				vars["strategy"] = strategy
			}

			return clientDoPrint(`
				mutation insertFibEntry($name: Name!, $nexthops: [ID!]!, $strategy: ID) {
					insertFibEntry(name: $name, nexthops: $nexthops, strategy: $strategy) {
						id
					}
				}
			`, vars, "insertFibEntry")
		},
	})
}

func init() {
	defineDeleteCommand("fib", "erase-fib", "Erase a FIB entry")
}
