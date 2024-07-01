package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chaseisabelle/flagz"
	"github.com/urfave/cli/v2"
)

func init() {
	var filename, name string
	var faces, ports flagz.Flagz
	var wantRX, wantTX, wantRxUnmatched bool
	var sampleProb float64
	var duration time.Duration

	type withID struct {
		ID string `json:"id"`
	}
	var writer string
	var sources []string
	createWriter := func(c *cli.Context) error {
		var result withID
		if e := clientDoPrint(c.Context, `
			mutation createPdumpWriter($filename: String!) {
				createPdumpWriter(filename: $filename) {
					filename
					id
					worker {
						id
						nid
						numaSocket
					}
				}
			}
		`, map[string]any{
			"filename": filename,
		}, "createPdumpWriter", &result); e != nil {
			return e
		}
		if cmdout {
			writer = "WRITER-ID"
		} else {
			writer = result.ID
		}
		return nil
	}
	createFaceSource := func(c *cli.Context, face, dir string) error {
		var result withID
		if e := clientDoPrint(c.Context, `
			mutation createPdumpFaceSource($writer: ID!, $face: ID!, $dir: PdumpDirection!, $name: Name!, $sampleProb: Float!) {
				createPdumpFaceSource(writer: $writer, face: $face, dir: $dir, names: [{ name: $name, sampleProbability: $sampleProb }]) {
					id
					face { id locator }
					dir
				}
			}
		`, map[string]any{
			"writer":     writer,
			"face":       face,
			"dir":        dir,
			"name":       name,
			"sampleProb": sampleProb,
		}, "createPdumpFaceSource", &result); e != nil {
			return e
		}
		if cmdout {
			sources = append(sources, fmt.Sprintf("FACE-SOURCE-ID:%s:%s", face, dir))
		} else {
			sources = append(sources, result.ID)
		}
		return nil
	}
	createEthPortSource := func(c *cli.Context, port, grab string) error {
		var result withID
		if e := clientDoPrint(c.Context, `
			mutation createPdumpEthPortSource($writer: ID!, $port: ID!, $grab: PdumpEthGrab!) {
				createPdumpEthPortSource(writer: $writer, port: $port, grab: $grab) {
					id
					port { id name macAddr }
					grab
				}
			}
		`, map[string]any{
			"writer": writer,
			"port":   port,
			"grab":   grab,
		}, "createPdumpEthPortSource", &result); e != nil {
			return e
		}
		if cmdout {
			sources = append(sources, fmt.Sprintf("ETHPORT-SOURCE-ID:%s:%s", port, grab))
		} else {
			sources = append(sources, result.ID)
		}
		return nil
	}
	waitFinish := func(*cli.Context) {
		if cmdout {
			if duration > 0 {
				fmt.Printf("sleep %0.1f\n", duration.Seconds())
			} else {
				fmt.Println("# traffic dumper running, continue below to stop")
			}
			fmt.Println()
		} else {
			if duration > 0 {
				time.Sleep(duration)
			} else {
				log.Print("traffic dumper running, press CTRL+C to stop")
				waitInterrupt()
			}
		}
	}
	closeAll := func(c *cli.Context) {
		for _, source := range sources {
			runDeleteCommand(c, source)
		}
		if writer != "" {
			runDeleteCommand(c, writer)
		}
	}

	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "filename",
			Usage:       "destination `filename`",
			Destination: &filename,
			Required:    true,
		},
		&cli.DurationFlag{
			Name:        "duration",
			Usage:       "packet dump duration",
			DefaultText: "interactive",
			Destination: &duration,
		},
	}

	defineCommand(&cli.Command{
		Category: "pdump",
		Name:     "pdump-face",
		Usage:    "Dump packet on a face",
		Flags: append([]cli.Flag{
			&cli.GenericFlag{
				Name:     "face",
				Usage:    "source face `ID` (repeatable)",
				Value:    &faces,
				Required: true,
			},
			&cli.BoolFlag{
				Name:        "rx",
				Usage:       "capture incoming packets",
				Value:       true,
				Destination: &wantRX,
			},
			&cli.BoolFlag{
				Name:        "tx",
				Usage:       "capture outgoing packets",
				Value:       true,
				Destination: &wantTX,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Value:       "/",
				Destination: &name,
			},
			&cli.Float64Flag{
				Name:        "sample-prob",
				Usage:       "sample `probability` between 0.0 and 1.0",
				Value:       1.0,
				Destination: &sampleProb,
			},
		}, commonFlags...),
		Action: func(c *cli.Context) error {
			defer closeAll(c)

			if e := createWriter(c); e != nil {
				return e
			}

			for _, face := range faces.Array() {
				for dir, enabled := range map[string]bool{"RX": wantRX, "TX": wantTX} {
					if !enabled {
						continue
					}
					if e := createFaceSource(c, face, dir); e != nil {
						return e
					}
				}
			}

			waitFinish(c)
			return nil
		},
	})

	defineCommand(&cli.Command{
		Category: "pdump",
		Name:     "pdump-ethport",
		Usage:    "Dump packet on an Ethernet port",
		Flags: append([]cli.Flag{
			&cli.GenericFlag{
				Name:     "port",
				Usage:    "source port `ID` (repeatable)",
				Value:    &ports,
				Required: true,
			},
			&cli.BoolFlag{
				Name:        "rx-unmatched",
				Usage:       "capture incoming packets not matching a face",
				Destination: &wantRxUnmatched,
				Required:    true,
			},
		}, commonFlags...),
		Action: func(c *cli.Context) error {
			defer closeAll(c)

			if e := createWriter(c); e != nil {
				return e
			}

			for _, port := range ports.Array() {
				for grab, enabled := range map[string]bool{"RxUnmatched": wantRxUnmatched} {
					if !enabled {
						continue
					}
					if e := createEthPortSource(c, port, grab); e != nil {
						return e
					}
				}
			}

			waitFinish(c)
			return nil
		},
	})
}
