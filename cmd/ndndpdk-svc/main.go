// Command ndndpdk-svc executes the NDN-DPDK service.
// It may be activated as a forwarder or a traffic generator.
package main

import (
	"errors"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

var (
	isActivated            int32
	errActivated           = errors.New("ndndpdk-svc is already activated")
	errActivateArgConflict = errors.New("exactly one activate argument should be specified")
)

type activator interface {
	Activate() error
}

func main() {
	rand.Seed(time.Now().UnixNano())

	gqlserver.AddMutation(&graphql.Field{
		Name:        "activate",
		Description: "Activate NDN-DPDK service.",
		Args: graphql.FieldConfigArgument{
			"forwarder": &graphql.ArgumentConfig{
				Description: "Activate as a forwarder.",
				Type:        gqlserver.JSON,
			},
			"trafficgen": &graphql.ArgumentConfig{
				Description: "Activate as a traffic generator.",
				Type:        gqlserver.JSON,
			},
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (result interface{}, e error) {
			if len(p.Args) != 1 {
				return nil, errActivateArgConflict
			}
			if !atomic.CompareAndSwapInt32(&isActivated, 0, 1) {
				return nil, errActivated
			}
			result = true

			tryActivate := func(key string, arg activator) {
				a, ok := p.Args[key]
				if !ok {
					return
				}

				if e = gqlserver.DecodeJSON(a, &arg); e != nil {
					return
				}

				log.Infof("activating %s", key)
				e = arg.Activate()

				if e != nil {
					go func() {
						time.Sleep(time.Second)
						log.WithError(e).Fatalf("activate %s error", key)
					}()
				}
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
			log.Info("shutdown requested")
			go func() {
				time.Sleep(time.Second)
				os.Exit(0)
			}()
			return true, nil
		},
	})

	gqlserver.Start()
	select {}
}
