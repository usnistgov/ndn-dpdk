// Command ndndpdk-upf runs a PFCP server that turns NDN-DPDK forwarder into 5G UPF.
package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/version"
	"go.uber.org/zap"
)

var (
	logger = logging.New("main")

	gqlCfg    gqlclient.Config
	upfParams upf.UpfParams

	client *gqlclient.Client
	theUPF *upf.UPF
)

var app = &cli.App{
	Version:              version.V.String(),
	Usage:                "Use NDN-DPDK as a UPF.",
	EnableBashCompletion: true,
	Flags: upfParams.DefineFlags([]cli.Flag{
		&cli.StringFlag{
			Name:        "gqlserver",
			Usage:       "GraphQL `endpoint` of NDN-DPDK service",
			Value:       "http://127.0.0.1:3030/",
			Destination: &gqlCfg.HTTPUri,
		},
	}),
	Action: func(c *cli.Context) (e error) {
		if client, e = gqlclient.New(gqlCfg); e != nil {
			return e
		}
		if e = upfParams.ProcessFlags(c); e != nil {
			return e
		}

		theUPF = upf.NewUPF(upfParams, createFace, destroyFace)
		return theUPF.Listen(c.Context)
	},
}

func main() {
	e := app.Run(os.Args)
	if e != nil {
		logger.Fatal("app exit", zap.Error(e))
	}
}
