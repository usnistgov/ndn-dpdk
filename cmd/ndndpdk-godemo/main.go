// Command ndndpdk-godemo demonstrates NDNgo library features.
package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/version"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

var (
	gqlserver string
	mtuFlag   int
	useNfd    bool
	enableLog bool

	client mgmt.Client
	face   mgmt.Face
	fwFace l3.FwFace
)

func openUplink(c *cli.Context) (e error) {
	switch client := client.(type) {
	case *gqlmgmt.Client:
		var loc memiftransport.Locator
		loc.Dataroom = mtuFlag
		face, e = client.OpenMemif(loc)
	default:
		face, e = client.OpenFace()
	}
	if e != nil {
		return e
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		return e
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Printf("uplink opened, state is %s", l3face.State())
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})
	return nil
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
			Usage:       "connect to NFD or YaNFD instead of NDN-DPDK (set FaceUri in NDN_CLIENT_TRANSPORT environment variable)",
			Destination: &useNfd,
		},
		&cli.BoolFlag{
			Name:        "logging",
			Usage:       "whether to enable logging",
			Value:       true,
			Destination: &enableLog,
		},
	},
	Before: func(c *cli.Context) (e error) {
		if !enableLog {
			log.SetOutput(io.Discard)
		}
		if useNfd {
			client, e = nfdmgmt.New()
		} else {
			client, e = gqlmgmt.New(gqlclient.Config{HTTPUri: gqlserver})
		}
		return e
	},
	After: func(c *cli.Context) (e error) {
		if face != nil {
			log.Printf("uplink closed, error is %v", face.Close())
		}
		return client.Close()
	},
}

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
}

func main() {
	sort.Sort(cli.CommandsByName(app.Commands))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	go func() {
		<-interrupt
		cancel()
	}()

	e := app.RunContext(ctx, os.Args)
	if e != nil {
		log.Fatal(e)
	}
}
