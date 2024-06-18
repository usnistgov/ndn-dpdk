// Command ndndpdk-upf runs a PFCP server that turns NDN-DPDK forwarder into 5G UPF.
package main

import (
	"net"
	"net/netip"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/version"
	"go.uber.org/zap"
)

var (
	logger = logging.New("main")

	gqlCfg gqlclient.Config
	upfCfg UpfConfig

	client   *gqlclient.Client
	pfcpConn *net.UDPConn
)

var app = &cli.App{
	Version:              version.V.String(),
	Usage:                "Use NDN-DPDK as a UPF.",
	EnableBashCompletion: true,
	Flags: upfCfg.DefineFlags([]cli.Flag{
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
		if e = upfCfg.ProcessFlags(c); e != nil {
			return e
		}

		pfcpAddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(upfCfg.UpfN4, 8805))
		if pfcpConn, e = net.ListenUDP("udp", pfcpAddr); e != nil {
			return e
		}
		return pfcpLoop(c.Context)
	},
}

func main() {
	e := app.Run(os.Args)
	if e != nil {
		logger.Fatal("app exit", zap.Error(e))
	}
}
