// Command ndndpdk-svc runs the NDN-DPDK service.
// It may be activated as a forwarder, a traffic generator, or a file server.
package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/graphql-go/graphql"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/version"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var logger = logging.New("main")

func init() {
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
}

func init() {
	type activator interface {
		Activate() error
	}
	var isActivated atomic.Bool

	gqlserver.AddMutation(&graphql.Field{
		Name: "activate",
		Description: "Activate NDN-DPDK service. " +
			"Exactly one argument must be provided.",
		Args: graphql.FieldConfigArgument{
			"forwarder": &graphql.ArgumentConfig{
				Description: "Activate as a forwarder. " +
					"This must be a JSON object that satisfies the schema given in 'forwarder.schema.json'.",
				Type: gqlserver.JSON,
			},
			"trafficgen": &graphql.ArgumentConfig{
				Description: "Activate as a traffic generator. " +
					"This must be a JSON object that satisfies the schema given in 'trafficgen.schema.json'.",
				Type: gqlserver.JSON,
			},
			"fileserver": &graphql.ArgumentConfig{
				Description: "Activate as a file server. " +
					"This must be a JSON object that satisfies the schema given in 'fileserver.schema.json'.",
				Type: gqlserver.JSON,
			},
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (result any, e error) {
			if len(p.Args) != 1 {
				return nil, errors.New("exactly one activate argument should be specified")
			}
			result = true

			tryActivate := func(role string, arg activator) {
				a, ok := p.Args[role]
				if !ok {
					return
				}
				if e = jsonhelper.Roundtrip(a, arg, jsonhelper.DisallowUnknownFields); e != nil {
					return
				}

				if !isActivated.CompareAndSwap(false, true) {
					e = errors.New("ndndpdk-svc is already activated")
					return
				}

				initXDPProgram()

				logEntry := logger.With(zap.String("role", role))
				logEntry.Info("activate start")
				if e = arg.Activate(); e != nil {
					delayedShutdown(func() { logEntry.Fatal("activate error", zap.Error(e)) })
					return
				}
				logEntry.Info("activate success")
			}

			tryActivate("forwarder", &fwArgs{})
			tryActivate("trafficgen", &genArgs{})
			tryActivate("fileserver", &fileServerArgs{})
			return
		},
	})
}

func init() {
	gqlserver.AddMutation(&graphql.Field{
		Name:        "shutdown",
		Description: "Shutdown NDN-DPDK service.",
		Args: graphql.FieldConfigArgument{
			"restart": &graphql.ArgumentConfig{
				Description: "Whether to restart the service.",
				Type:        graphql.Boolean,
			},
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			restart, ok := p.Args["restart"].(bool)
			if !ok {
				restart = false
			}
			exitCode := 0
			if restart {
				exitCode = 75
			}

			logger.Info("shutdown requested by GraphQL", zap.Bool("restart", restart))
			daemon.SdNotify(false, daemon.SdNotifyStopping)
			delayedShutdown(func() { os.Exit(exitCode) })
			return true, nil
		},
	})
}

var app = &cli.App{
	Version: version.V.String(),
	Usage:   "Provide NDN-DPDK service.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "gqlserver",
			Usage: "GraphQL HTTP server base URI",
			Value: "http://127.0.0.1:3030/",
		},
	},
	Action: func(c *cli.Context) (e error) {
		listen, e := gqlclient.MakeListenAddress(c.String("gqlserver"))
		if e != nil {
			return cli.Exit(e, 1)
		}

		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, unix.SIGINT, unix.SIGTERM)
			sig := <-c
			logger.Info("shutdown requested by signal", zap.Stringer("signal", sig))
			delayedShutdown(func() { os.Exit(0) })
		}()

		go systemdNotify()

		gqlserver.Prepare()
		logger.Info("GraphQL HTTP server starting", zap.String("listen", listen))
		return cli.Exit(http.ListenAndServe(listen, nil), 1)
	},
}

func main() {
	var uname unix.Utsname
	unix.Uname(&uname)
	logger.Info("NDN-DPDK service starting",
		zap.Any("version", version.V),
		zap.Int("uid", os.Getuid()),
		zap.ByteString("linux", bytes.TrimRight(uname.Release[:], string([]byte{0}))),
		zap.String("dpdk", eal.Version),
		zap.String("spdk", spdkenv.Version),
	)

	app.Run(os.Args)
}

func systemdNotify() {
	daemon.SdNotify(false, daemon.SdNotifyReady)

	d, e := daemon.SdWatchdogEnabled(false)
	if d == 0 || e != nil {
		logger.Debug("systemd watchdog not configured", zap.Error(e))
		return
	}

	d /= 2
	logger.Debug("systemd watchdog enabled", zap.Duration("duration", d))
	for range time.Tick(d) {
		daemon.SdNotify(false, daemon.SdNotifyWatchdog)
	}
}
