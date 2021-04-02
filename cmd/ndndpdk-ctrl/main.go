// Command ndndpdk-ctrl controls the NDN-DPDK service.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"

	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/mk/version"
)

var (
	gqlserver string
	cmdout    bool
	client    *gqlclient.Client
)

func clientDoPrint(query string, vars interface{}, key string) error {
	if cmdout {
		gqArgs := []string{gqlserver, "-q", query}
		if vars != nil {
			j, e := json.MarshalIndent(vars, "", "  ")
			if e != nil {
				return e
			}
			gqArgs = append(gqArgs, "--variablesJSON", string(j))
		}
		jqArgs := []string{"-c"}
		if key == "" {
			jqArgs = append(jqArgs, ".data")
		} else {
			jqArgs = append(jqArgs, ".data."+key)
		}
		fmt.Println("gq", shellquote.Join(gqArgs...), "|", "jq", shellquote.Join(jqArgs...))
		return nil
	}

	var value interface{}
	e := client.Do(query, vars, key, &value)
	if e != nil {
		return e
	}

	if val := reflect.ValueOf(value); val.Kind() == reflect.Slice {
		for i, last := 0, val.Len(); i < last; i++ {
			j, _ := json.Marshal(val.Index(i).Interface())
			fmt.Println(string(j))
		}
	} else {
		j, _ := json.Marshal(value)
		fmt.Println(string(j))
	}
	return nil
}

var app = &cli.App{
	Version: version.Get().String(),
	Usage:   "Control NDN-DPDK service.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Value:       "http://127.0.0.1:3030/",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Destination: &gqlserver,
		},
		&cli.BoolFlag{
			Name:        "cmdout",
			Value:       false,
			Usage:       "print command line instead of executing",
			Destination: &cmdout,
		},
	},
	Before: func(c *cli.Context) (e error) {
		if cmdout {
			return nil
		}
		client, e = gqlclient.New(gqlserver)
		return e
	},
}

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
}

func defineDeleteCommand(category, commandName, usage string) {
	var id string

	defineCommand(&cli.Command{
		Category: category,
		Name:     commandName,
		Usage:    usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "object `ID`",
				Destination: &id,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation delete($id: ID!) {
					delete(id: $id)
				}
			`, map[string]interface{}{
				"id": id,
			}, "delete")
		},
	})
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
			return clientDoPrint(`
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
