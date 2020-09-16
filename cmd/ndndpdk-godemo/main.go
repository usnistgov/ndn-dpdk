// Command ndndpdk-godemo demonstrates NDNgo library features.
package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/mk/version"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
)

var (
	interrupt = make(chan os.Signal, 1)
	client    mgmt.Client
	face      mgmt.Face
	fwFace    l3.FwFace
)

func openUplink(c *cli.Context) (e error) {
	if face, e = client.OpenFace(); e != nil {
		return e
	}
	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(face.Face()); e != nil {
		return e
	}
	fw.AddReadvertiseDestination(face)
	log.Print("uplink opened")
	return nil
}

var app = &cli.App{
	Version: version.Get().String(),
	Usage:   "NDNgo library demo.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "gqlserver",
			Value:   "http://127.0.0.1:3030/",
			Usage:   "GraphQL `endpoint` of NDN-DPDK daemon.",
			EnvVars: []string{"GQLSERVER"},
		},
	},
	Before: func(c *cli.Context) (e error) {
		signal.Notify(interrupt, syscall.SIGINT)
		client, e = gqlmgmt.New(c.String("gqlserver"))
		return e
	},
	After: func(c *cli.Context) (e error) {
		if face != nil {
			face.Close()
			log.Print("uplink closed")
		}
		return client.Close()
	},
}

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	sort.Sort(cli.CommandsByName(app.Commands))
	e := app.Run(os.Args)
	if e != nil {
		log.Fatal(e)
	}
}
