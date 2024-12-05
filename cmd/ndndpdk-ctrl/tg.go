package main

import (
	"fmt"
	"math"
	"time"

	"github.com/urfave/cli/v2"
)

func init() {
	defineStdinJSONCommand(stdinJSONCommand{
		Category:   "trafficgen",
		Name:       "start-trafficgen",
		Usage:      "Start a traffic generator",
		SchemaName: "gen",
		ParamNoun:  "traffic patterns",
		Action: func(c *cli.Context, arg map[string]any) error {
			return clientDoPrint(c.Context, `
				mutation startTrafficGen(
					$face: JSON!
					$producer: TgpConfigInput
					$fileServer: FileServerConfigInput
					$consumer: TgcConfigInput
					$fetcher: FetcherConfigInput
				) {
					startTrafficGen(
						face: $face
						producer: $producer
						fileServer: $fileServer
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
	var withPerPattern bool
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
			&cli.BoolFlag{
				Name:        "per-pattern",
				Usage:       "with per-pattern counters",
				Destination: &withPerPattern,
				Value:       false,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				subscription watchTrafficGen($id: ID!, $interval: NNNanoseconds!, $withPerPattern: Boolean!) {
					tgCounters(id: $id, interval: $interval) {
						producer {
							nInterests
							nNoMatch
							nAllocError
							perPattern @include(if: $withPerPattern) {
								nInterests
								perReply
							}
						}
						fileServer {
							reqRead
							reqLs
							reqMetadata
							fdNew
							fdNotFound
							fdUpdateStat
							fdClose
							uringAllocErrs
							uringSubmitted
							uringSubmitNonBlock
							uringSubmitWait
							cqeFail
						}
						consumer {
							nInterests
							nData
							nNacks
							nAllocError
							perPattern @include(if: $withPerPattern) {
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
			`, map[string]any{
				"id":             id,
				"interval":       interval.Nanoseconds(),
				"withPerPattern": withPerPattern,
			}, "tgCounters")
		},
	})
}

func init() {
	var fetcher, name, filename string
	var segmentBegin, segmentEnd uint64
	var fileSize int64
	var segmentLen int
	defineCommand(&cli.Command{
		Category: "trafficgen",
		Name:     "start-fetch",
		Usage:    "Start fetching a segmented object",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "fetcher",
				Usage:       "fetcher `ID`",
				Destination: &fetcher,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.Uint64Flag{
				Name:        "segment-begin",
				Usage:       "first segment `number` (inclusive)",
				Destination: &segmentBegin,
				Value:       0,
			},
			&cli.Uint64Flag{
				Name:        "segment-end",
				Usage:       "last segment `number` (exclusive)",
				Destination: &segmentEnd,
				Value:       math.MaxUint64,
			},
			&cli.StringFlag{
				Name:        "filename",
				Usage:       "output file name",
				DefaultText: "not saving to file",
				Destination: &filename,
			},
			&cli.Int64Flag{
				Name:        "file-size",
				Usage:       "file size `octets`",
				Destination: &fileSize,
			},
			&cli.IntFlag{
				Name:        "segment-len",
				Usage:       "segment length `octets`",
				Destination: &segmentLen,
			},
		},
		Action: func(c *cli.Context) error {
			task := map[string]any{
				"prefix": name,
			}
			if c.IsSet("segment-begin") {
				task["segmentBegin"] = segmentBegin
			}
			if c.IsSet("segment-end") {
				task["segmentEnd"] = segmentEnd
			}
			if filename != "" {
				task["filename"] = filename
				task["fileSize"] = fileSize
				task["segmentLen"] = segmentLen
			}
			return clientDoPrint(c.Context, `
				mutation fetch($fetcher: ID!, $task: FetchTaskDefInput!) {
					fetch(fetcher: $fetcher, task: $task) {
						id
						task {
							prefix
							interestLifetime
							hopLimit
							segmentBegin
							segmentEnd
							filename
							fileSize
							segmentLen
						}
						worker {
							id
							nid
							numaSocket
						}
					}
				}
			`, map[string]any{
				"fetcher": fetcher,
				"task":    task,
			}, "fetch")
		},
	})
}

func init() {
	defineDeleteCommand("trafficgen", "stop-fetch", "Stop fetching a segmented object", "fetch task")
}

func init() {
	var id string
	var interval time.Duration
	var autoStop bool
	defineCommand(&cli.Command{
		Category: "trafficgen",
		Name:     "watch-fetch",
		Usage:    "Watch fetch task progress",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       "fetch task `ID`",
				Destination: &id,
				Required:    true,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "update `interval`",
				Destination: &interval,
				Value:       time.Second,
			},
			&cli.BoolFlag{
				Name:        "auto-stop",
				Usage:       "automatically stop fetch task upon finishing",
				Destination: &autoStop,
				Value:       false,
			},
		},
		Action: func(c *cli.Context) error {
			handleUpdate := func(update struct {
				Finished *time.Duration `json:"finished"`
			}) bool {
				return update.Finished == nil
			}
			if !autoStop {
				handleUpdate = nil
			}

			if e := clientDoPrint(c.Context, `
				subscription watchFetch($id: ID!, $interval: NNNanoseconds!) {
					fetchCounters(id: $id, interval: $interval)
				}
			`, map[string]any{
				"id":       id,
				"interval": interval.Nanoseconds(),
			}, "fetchCounters", handleUpdate); e != nil {
				return e
			}
			if !autoStop {
				return nil
			}

			if cmdout {
				fmt.Println("# wait until .finished becomes non-null, continue below to stop")
				fmt.Println()
			}
			return runDeleteCommand(c, id)
		},
	})
}
