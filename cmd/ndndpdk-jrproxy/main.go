// Command ndndpdk-jrproxy exposes NDN-DPDK GraphQL API as JSON-RPC 2.0 management API (2019).
package main

import (
	"log"
	"net"
	"net/rpc"
	"os"

	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/version"
)

var client *gqlclient.Client

func main() {
	var gqlserver string
	var tcpListen string
	app := &cli.App{
		Version: version.V.String(),
		Usage:   "NDN-DPDK JSON-RPC 2.0 management proxy.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gqlserver",
				Value:       "http://127.0.0.1:3030/",
				Usage:       "GraphQL `endpoint` of NDN-DPDK service",
				Destination: &gqlserver,
			},
			&cli.StringFlag{
				Name:        "listen",
				Usage:       "TCP listen `endpoint`",
				Value:       "127.0.0.1:6345",
				Destination: &tcpListen,
			},
		},
		Action: func(c *cli.Context) (e error) {
			client, e = gqlclient.New(gqlclient.Config{HTTPUri: gqlserver})
			if e != nil {
				return e
			}

			listener, e := net.Listen("tcp", tcpListen)
			if e != nil {
				return e
			}

			for {
				conn, e := listener.Accept()
				if e != nil {
					return e
				}
				go jsonrpc2.ServeConn(conn)
			}
		},
	}
	e := app.Run(os.Args)
	if e != nil {
		log.Fatal(e)
	}
}

func init() {
	rpc.Register(Version{})
	rpc.Register(Face{})
	rpc.Register(Fib{})
}
