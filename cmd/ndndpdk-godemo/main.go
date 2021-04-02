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
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
	"go4.org/must"
)

var (
	interrupt = make(chan os.Signal, 1)
	client    *gqlmgmt.Client
	face      mgmt.Face
	fwFace    l3.FwFace

	gqlserverFlag string
	mtuFlag       int
)

func openUplink(c *cli.Context) (e error) {
	var loc memiftransport.Locator
	loc.Dataroom = mtuFlag
	if face, e = client.OpenMemif(loc); e != nil {
		return e
	}

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(face.Face()); e != nil {
		return e
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Print("uplink opened")
	return nil
}

var app = &cli.App{
	Version: version.Get().String(),
	Usage:   "NDNgo library demo.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Value:       "http://127.0.0.1:3030/",
			Destination: &gqlserverFlag,
		},
		&cli.IntFlag{
			Name:        "mtu",
			Usage:       "application face `MTU`",
			Destination: &mtuFlag,
		},
	},
	Before: func(c *cli.Context) (e error) {
		signal.Notify(interrupt, syscall.SIGINT)
		client, e = gqlmgmt.New(gqlserverFlag)
		return e
	},
	After: func(c *cli.Context) (e error) {
		if face != nil {
			must.Close(face)
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
