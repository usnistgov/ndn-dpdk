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
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/version"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
	"go4.org/must"
)

var (
	interrupt = make(chan os.Signal, 1)
	client    mgmt.Client
	face      mgmt.Face
	fwFace    l3.FwFace

	gqlserver string
	mtuFlag   int
	useNfd    bool
)

func openUplink(c *cli.Context) (e error) {
	if gqlClient, ok := client.(*gqlmgmt.Client); ok {
		var loc memiftransport.Locator
		loc.Dataroom = mtuFlag
		face, e = gqlClient.OpenMemif(loc)
	} else {
		face, e = client.OpenFace()
	}
	if e != nil {
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

func onInterrupt(cancel func()) {
	go func() {
		<-interrupt
		cancel()
	}()
}

var app = &cli.App{
	Version:              version.V.String(),
	Usage:                "NDNgo library demo.",
	EnableBashCompletion: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Value:       "http://127.0.0.1:3030/",
			Destination: &gqlserver,
		},
		&cli.IntFlag{
			Name:        "mtu",
			Usage:       "application face `MTU`",
			Destination: &mtuFlag,
		},
		&cli.BoolFlag{
			Name:        "nfd",
			Usage:       "connect to NFD or YaNFD (set FaceUri in NDN_CLIENT_TRANSPORT environment variable)",
			Destination: &useNfd,
		},
	},
	Before: func(c *cli.Context) (e error) {
		signal.Notify(interrupt, syscall.SIGINT)
		if useNfd {
			client, e = nfdmgmt.New()
		} else {
			if os.Getuid() != 0 {
				log.Print("running as non-root, some features will not work")
			}
			client, e = gqlmgmt.New(gqlclient.Config{HTTPUri: gqlserver})
		}
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
