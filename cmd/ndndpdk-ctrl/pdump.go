package main

import (
	"fmt"
	"log"
	"time"

	"github.com/urfave/cli/v2"
)

func init() {
	var filename, name string
	var faces cli.StringSlice
	var wantRX, wantTX bool
	var sampleProb float64
	var duration time.Duration

	type withID struct {
		ID string `json:"id"`
	}
	var writer string
	var faceSources []string
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
			`, map[string]interface{}{
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
		`, map[string]interface{}{
			"writer":     writer,
			"face":       face,
			"dir":        dir,
			"name":       name,
			"sampleProb": sampleProb,
		}, "createPdumpFaceSource", &result); e != nil {
			return e
		}
		if cmdout {
			faceSources = append(faceSources, fmt.Sprintf("FACE-SOURCE-ID:%s:%s", face, dir))
		} else {
			faceSources = append(faceSources, result.ID)
		}
		return nil
	}
	closeAll := func(c *cli.Context) {
		for _, faceSource := range faceSources {
			runDeleteCommand(c, faceSource)
		}
		if writer != "" {
			runDeleteCommand(c, writer)
		}
	}

	defineCommand(&cli.Command{
		Category: "pdump",
		Name:     "pdump-face",
		Usage:    "Dump packet on a face",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "filename",
				Usage:       "destination `filename`",
				Destination: &filename,
				Required:    true,
			},
			&cli.StringSliceFlag{
				Name:        "face",
				Usage:       "source face `ID` (repeatable)",
				Destination: &faces,
				Required:    true,
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
			&cli.DurationFlag{
				Name:        "duration",
				Usage:       "packet dump duration",
				DefaultText: "interactive",
				Destination: &duration,
			},
		},
		Action: func(c *cli.Context) error {
			defer closeAll(c)

			if e := createWriter(c); e != nil {
				return e
			}

			for _, face := range faces.Value() {
				for dir, enabled := range map[string]bool{"RX": wantRX, "TX": wantTX} {
					if !enabled {
						continue
					}
					if e := createFaceSource(c, face, dir); e != nil {
						return e
					}
				}
			}

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
					<-interrupt
				}
			}

			return nil
		},
	})
}
