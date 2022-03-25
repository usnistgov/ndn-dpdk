package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

func init() {
	defineCommand(&cli.Command{
		Category: "fib",
		Name:     "list-fib",
		Usage:    "List FIB entries",
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
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
	var name, strategy, params string
	var nexthops cli.StringSlice

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
			&cli.StringFlag{
				Name:        "params",
				Usage:       "forwarding strategy parameters `JSON`",
				Destination: &params,
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
			if params != "" {
				var paramsJ map[string]interface{}
				if e := json.Unmarshal([]byte(params), &paramsJ); e != nil {
					return fmt.Errorf("params: %w", e)
				}
				vars["params"] = paramsJ
			}

			return clientDoPrint(c.Context, `
				mutation insertFibEntry($name: Name!, $nexthops: [ID!]!, $strategy: ID, $params: JSON) {
					insertFibEntry(name: $name, nexthops: $nexthops, strategy: $strategy, params: $params) {
						id
					}
				}
			`, vars, "insertFibEntry")
		},
	})
}

func init() {
	defineDeleteCommand("fib", "erase-fib", "Erase a FIB entry", "FIB entry")
}
