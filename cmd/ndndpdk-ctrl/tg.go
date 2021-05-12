package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

func init() {
	defineStdinJSONCommand(stdinJSONCommand{
		Category:   "trafficgen",
		Name:       "start-trafficgen",
		Usage:      "Start a traffic generator",
		SchemaName: "gen",
		Action: func(c *cli.Context, arg map[string]interface{}) error {
			return clientDoPrint(c.Context, `
				mutation startTrafficGen(
					$face: JSON!
					$producer: TgProducerConfigInput
					$consumer: TgConsumerConfigInput
					$fetcher: FetcherConfigInput
				) {
					startTrafficGen(
						face: $face
						producer: $producer
						consumer: $consumer
						fetcher: $fetcher
					) {
						id
						face { id }
						producer { id }
						consumer { id }
						fetcher { id }
					}
				}
			`, arg, "startTrafficGen")
		},
	})
}

func init() {
	defineDeleteCommand("trafficgen", "stop-trafficgen", "Stop a traffic generator", "traffic generator")
}

func init() {
	var id string
	var interval time.Duration
	defineCommand(&cli.Command{
		Category: "trafficgen",
		Name:     "watch-trafficgen",
		Usage:    "Watch traffic generator counters",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "traffic generator `ID`",
				Destination: &id,
				Required:    true,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "update `interval`",
				Destination: &interval,
				Value:       time.Second,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				subscription watchTrafficGen($id: ID!, $interval: NNNanoseconds!) {
					tgCounters(id: $id, interval: $interval) {
						producer {
							nInterests
							nNoMatch
							nAllocError
							perPattern {
								nInterests
								perReply
							}
						}
						consumer {
							nInterests
							nData
							nNacks
							nAllocError
							perPattern {
								nInterests
								nData
								nNacks
							}
							rtt {
								mean
								stdev
							}
						}
					}
				}
			`, map[string]interface{}{
				"id":       id,
				"interval": interval.Nanoseconds(),
			}, "tgCounters")
		},
	})
}
