package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"go4.org/must"
)

func init() {
	var name string
	var wantAdvertise bool
	var payloadLen int
	defineCommand(&cli.Command{
		Name:  "pingserver",
		Usage: "Reachability test server: reply to every Interest under a prefix.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Producer name `prefix`.",
				Destination: &name,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "advertise",
				Usage:       "Whether to advertise/register prefix.",
				Value:       true,
				Destination: &wantAdvertise,
			},
			&cli.IntFlag{
				Name:        "payload",
				Usage:       "Payload length.",
				Destination: &payloadLen,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			payload := make([]byte, payloadLen)
			rand.Read(payload)
			p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
				Prefix:      ndn.ParseName(name),
				NoAdvertise: !wantAdvertise,
				Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
					log.Print(interest)
					return ndn.MakeData(interest, payload), nil
				},
			})
			if e != nil {
				return e
			}
			<-interrupt
			must.Close(p)
			return nil
		},
	})
}

func init() {
	var name string
	var interval, lifetime time.Duration
	defineCommand(&cli.Command{
		Name:  "pingclient",
		Usage: "Reachability test client: periodically send Interest under a prefix.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Consumer name `prefix`.",
				Destination: &name,
				Required:    true,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "The `interval` between Interests.",
				Value:       100 * time.Millisecond,
				Destination: &interval,
			},
			&cli.DurationFlag{
				Name:        "lifetime",
				Usage:       "Interest `lifetime`.",
				Value:       1000 * time.Millisecond,
				Destination: &lifetime,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			seqNum := rand.Uint64()
			var nData, nErrors int64
			for {
				select {
				case <-interrupt:
					return nil
				case timestamp := <-ticker.C:
					go func(t0 time.Time, s uint64) {
						interest := ndn.MakeInterest(fmt.Sprintf("%s/%016X", name, seqNum), ndn.MustBeFreshFlag, lifetime)
						_, e := endpoint.Consume(ctx, interest, endpoint.ConsumerOptions{})
						rtt := time.Since(t0)
						if e == nil {
							atomic.AddInt64(&nData, 1)
							log.Printf("%6.2f%% D %016X %6dus", 100*float64(nData)/float64(nData+nErrors), seqNum, rtt.Microseconds())
						} else {
							atomic.AddInt64(&nErrors, 1)
							log.Printf("%6.2f%% E %016X %v", 100*float64(nData)/float64(nData+nErrors), seqNum, e)
						}
					}(timestamp, seqNum)
					seqNum++
				}
			}
		},
	})
}
