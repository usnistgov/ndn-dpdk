package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/mk/version"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
)

var gqlserver string
var client *gqlmgmt.Client

func clientDoPrint(query string, vars interface{}, key string) error {
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
	Usage:   "Control NDN-DPDK daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Value:       "http://127.0.0.1:3030/",
			Usage:       "GraphQL API of NDN-DPDK daemon",
			EnvVars:     []string{"GQLSERVER"},
			Destination: &gqlserver,
		},
	},
	Before: func(c *cli.Context) (e error) {
		client, e = gqlmgmt.New(gqlserver)
		return e
	},
}

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
}

func init() {
	var id string

	defineCommand(&cli.Command{
		Name:    "delete",
		Aliases: []string{"destroy-face"},
		Usage:   "Delete object.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "Object ID.",
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
	e := app.Run(os.Args)
	if e != nil {
		log.Fatal(e)
	}
}
