// Command ndndpdk-jrproxy exposes NDN-DPDK GraphQL API as JSON-RPC 2.0 management API (2019).
package main

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
)

var client *gqlclient.Client

func main() {
	var gqlserver string
	var tcpListen string
	app := &cli.App{
		Name: "ndndpdk-ctrl",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gqlserver",
				EnvVars:     []string{"GQLSERVER"},
				Value:       "http://127.0.0.1:3030/",
				Usage:       "GraphQL `endpoint` of NDN-DPDK daemon",
				Destination: &gqlserver,
			},
			&cli.StringFlag{
				Name:        "listen",
				EnvVars:     []string{"MGMT"},
				Usage:       "TCP listen `endpoint`",
				Value:       "127.0.0.1:6345",
				Destination: &tcpListen,
			},
			&cli.IntFlag{
				Name:        "face-create-mtu",
				Usage:       "Override `MTU` in Face.Create command.",
				Destination: &faceCreateMTU,
			},
		},
		Action: func(c *cli.Context) (e error) {
			client, e = gqlclient.New(gqlserver)
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
					if ne, ok := e.(net.Error); ok && ne.Temporary() {
						continue
					} else {
						return e
					}
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
	rand.Seed(time.Now().UnixNano())
	rpc.Register(Version{})
	rpc.Register(Face{})
	rpc.Register(Fib{})
}
