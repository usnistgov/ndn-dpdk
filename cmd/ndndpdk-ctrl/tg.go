package main

import "github.com/urfave/cli/v2"

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

	defineDeleteCommand("trafficgen", "stop-trafficgen", "Stop a traffic generator", "traffic generator")
}
