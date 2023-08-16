package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func defineFetchOptionsFlags(fetchOptions *segmented.FetchOptions, flags []cli.Flag) []cli.Flag {
	return append(flags,
		&cli.IntFlag{
			Name:        "retx-limit",
			Usage:       "retransmission limit",
			Destination: &fetchOptions.RetxLimit,
			Value:       15,
		},
		&cli.IntFlag{
			Name:        "max-cwnd",
			Usage:       "maximum congestion window",
			Destination: &fetchOptions.MaxCwnd,
			Value:       24,
		},
	)
}

func retrieveSegmented(ctx context.Context, name ndn.Name, filename string, segmentLen int, fetchOptions segmented.FetchOptions) (e error) {
	out := os.Stdout
	if filename != "" {
		file, e := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o666)
		if e != nil {
			return e
		}
		defer file.Close()
		out = file
	}

	fetcher := segmented.Fetch(name, fetchOptions)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cnt, total := fetcher.Count(), fetcher.EstimatedTotal()
				if total <= 0 {
					log.Printf("retrieved %d segments, total unknown", cnt)
				} else {
					log.Printf("retrieved %d of %d segments (%0.2f%%)", cnt, total, 100*float64(cnt)/float64(total))
				}
			}
		}
	}()

	t0 := time.Now()
	e = fetcher.Pipe(ctx, out)
	if e == nil {
		log.Printf("finished %d segments in %v", fetcher.Count(), time.Since(t0).Truncate(time.Millisecond))
	}
	return e
}

func init() {
	var name, file string
	var serveOptions segmented.ServeOptions
	defineCommand(&cli.Command{
		Name:  "put",
		Usage: "Publish segmented object.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "file",
				Usage:       "filename (must be regular file)",
				Destination: &file,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "chunk-size",
				Usage:       "segment payload `size`",
				Destination: &serveOptions.ChunkSize,
				Value:       4096,
			},
		},
		Before: openUplink,
		Action: func(c *cli.Context) error {
			serveOptions.Prefix = ndn.ParseName(name)

			f, e := os.Open(file)
			if e != nil {
				log.Fatal(e)
			}
			defer f.Close()

			_, e = segmented.Serve(c.Context, f, serveOptions)
			if e != nil {
				return e
			}

			<-c.Context.Done()
			return nil
		},
	})
}

func init() {
	var name, filename string
	var fetchOptions segmented.FetchOptions
	defineCommand(&cli.Command{
		Name:  "get",
		Usage: "Retrieve segmented object.",
		Flags: defineFetchOptionsFlags(&fetchOptions, []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name `prefix`",
				Destination: &name,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "filename",
				Usage:       "output file name",
				DefaultText: "write to stdout",
				Destination: &filename,
			},
		}),
		Before: openUplink,
		Action: func(c *cli.Context) error {
			return retrieveSegmented(c.Context, ndn.ParseName(name), filename, 0, fetchOptions)
		},
	})
}
