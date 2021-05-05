package main

import (
	"encoding/base64"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

func init() {
	var withFib bool

	defineCommand(&cli.Command{
		Category: "strategy",
		Name:     "list-strategy",
		Aliases:  []string{"list-strategies"},
		Usage:    "List strategies",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "fib",
				Usage:       "show FIB entries",
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

func init() {
	var name string
	var elffile string
	var elf []byte

	defineCommand(&cli.Command{
		Category: "strategy",
		Name:     "load-strategy",
		Usage:    "Load a strategy ELF program",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "short `name`",
				Destination: &name,
			},
			&cli.StringFlag{
				Name:        "elffile",
				Usage:       "ELF program `file`",
				Destination: &elffile,
				Required:    true,
			},
		},
		Before: func(c *cli.Context) (e error) {
			elf, e = os.ReadFile(elffile)
			if e != nil {
				return e
			}
			if name == "" {
				name = path.Base(elffile)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation loadStrategy($name: String!, $elf: Bytes!) {
					loadStrategy(name: $name, elf: $elf) {
						id
						name
					}
				}
			`, map[string]interface{}{
				"name": name,
				"elf":  base64.StdEncoding.EncodeToString(elf),
			}, "loadStrategy")
		},
	})
}

func init() {
	defineDeleteCommand("strategy", "unload-strategy", "Unload a strategy ELF program", "strategy program")
}
