// Command ndndpdk-svc executes the NDN-DPDK service.
// It may be activated as a forwarder or a traffic generator.
package main

import (
	"errors"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/graphql-go/graphql"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/mk/version"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var logger = logging.New("main")

var (
	isActivated            int32
	errActivated           = errors.New("ndndpdk-svc is already activated")
	errActivateArgConflict = errors.New("exactly one activate argument should be specified")
)

type activator interface {
	Activate() error
}

func init() {
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})

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
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (result interface{}, e error) {
			if len(p.Args) != 1 {
				return nil, errActivateArgConflict
			}
			result = true

			tryActivate := func(key string, arg activator) {
				a, ok := p.Args[key]
				if !ok {
					return
				}
				if e = jsonhelper.Roundtrip(a, arg, jsonhelper.DisallowUnknownFields); e != nil {
					return
				}

				if !atomic.CompareAndSwapInt32(&isActivated, 0, 1) {
					e = errActivated
					return
				}

				initXDPProgram()

				logEntry := logger.With(zap.String("role", key))
				logEntry.Info("activate start")
				if e = arg.Activate(); e != nil {
					delayedShutdown(func() { logEntry.Fatal("activate error", zap.Error(e)) })
					return
				}
				logEntry.Info("activate success")
			}

			tryActivate("forwarder", &fwArgs{})
			tryActivate("trafficgen", &genArgs{})
			return
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "shutdown",
		Description: "Shutdown NDN-DPDK service.",
		Type:        gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			logger.Info("shutdown requested")
			daemon.SdNotify(false, daemon.SdNotifyStopping)
			delayedShutdown(func() { os.Exit(0) })
			return true, nil
		},
	})
}

var app = &cli.App{
	Version: version.Get().String(),
	Usage:   "Provide NDN-DPDK service.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Usage: "GraphQL HTTP server `host:port`",
			Value: "127.0.0.1:3030",
		},
	},
	Action: func(c *cli.Context) (e error) {
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, unix.SIGINT, unix.SIGTERM)
			<-c
			delayedShutdown(func() { os.Exit(0) })
		}()

		go systemdNotify()

		gqlserver.Prepare()
		listen := c.String("listen")
		logger.Info("GraphQL HTTP server starting",
			zap.String("listen", listen),
		)
		return cli.Exit(http.ListenAndServe(listen, nil), 1)
	},
}

func main() {
	rand.Seed(time.Now().UnixNano())
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
