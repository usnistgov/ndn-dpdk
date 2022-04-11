package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"go.uber.org/atomic"
)

func init() {
	var name string
	var wantAdvertise, wantSign bool
	var payloadLen int
	defineCommand(&cli.Command{
		Name:  "pingserver",
		Usage: "Reachability test server: reply to every Interest under a prefix.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "producer name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "advertise",
				Usage:       "whether to advertise/register prefix",
				Value:       true,
				Destination: &wantAdvertise,
			},
			&cli.IntFlag{
				Name:        "payload",
				Usage:       "payload length",
				Destination: &payloadLen,
			},
			&cli.BoolFlag{
				Name:        "signed",
				Usage:       "enable packet signing (SigSha256)",
				Destination: &wantSign,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			payload := make([]byte, payloadLen)
			rand.Read(payload)
			var signer ndn.Signer
			if wantSign {
				signer = ndn.DigestSigning
			}

			ctx, cancel := context.WithCancel(context.Background())
			onInterrupt(cancel)

			_, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
				Prefix:      ndn.ParseName(name),
				NoAdvertise: !wantAdvertise,
				Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
					log.Print(interest)
					return ndn.MakeData(interest, payload), nil
				},
				DataSigner: signer,
			})
			if e != nil {
				return e
			}

			<-ctx.Done()
			return nil
		},
	})
}

func init() {
	var name string
	var interval, lifetime time.Duration
	var wantVerify bool
	defineCommand(&cli.Command{
		Name:  "pingclient",
		Usage: "Reachability test client: periodically send Interest under a prefix.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "consumer name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "the `interval` between Interests",
				Value:       100 * time.Millisecond,
				Destination: &interval,
			},
			&cli.DurationFlag{
				Name:        "lifetime",
				Usage:       "Interest `lifetime`",
				Value:       1000 * time.Millisecond,
				Destination: &lifetime,
			},
			&cli.BoolFlag{
				Name:        "verified",
				Usage:       "enable packet verification (SigSha256)",
				Destination: &wantVerify,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			var verifier ndn.Verifier
			if wantVerify {
				verifier = ndn.DigestSigning
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			seqNum := rand.Uint64()
			var nData, nErrors atomic.Int64
			for {
				select {
				case <-interrupt:
					return nil
				case timestamp := <-ticker.C:
					go func(t0 time.Time, s uint64) {
						interest := ndn.MakeInterest(fmt.Sprintf("%s/%016X", name, seqNum), ndn.MustBeFreshFlag, lifetime)
						_, e := endpoint.Consume(ctx, interest, endpoint.ConsumerOptions{
							Verifier: verifier,
						})
						rtt := time.Since(t0)
						if e == nil {
							nDataL, nErrorsL := nData.Inc(), nErrors.Load()
							log.Printf("%6.2f%% D %016X %6dus", 100*float64(nDataL)/float64(nDataL+nErrorsL), seqNum, rtt.Microseconds())
						} else {
							nDataL, nErrorsL := nData.Load(), nErrors.Inc()
							log.Printf("%6.2f%% E %016X %v", 100*float64(nDataL)/float64(nDataL+nErrorsL), seqNum, e)
						}
					}(timestamp, seqNum)
					seqNum++
				}
			}
		},
	})
}
