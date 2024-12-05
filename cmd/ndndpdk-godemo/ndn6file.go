package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr/ndn6file"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func retrieveFileMetadata(ctx context.Context, name string, fetchOptions segmented.FetchOptions) (m ndn6file.Metadata, e error) {
	e = rdr.RetrieveMetadata(ctx, &m, ndn.ParseName(name), endpoint.ConsumerOptions{
		Retx: endpoint.RetxOptions{Limit: fetchOptions.RetxLimit},
	})
	if e != nil {
		return
	}
	log.Printf("retrieved metadata %s", m.Name)
	return
}

func init() {
	var name string
	var fetchOptions segmented.FetchOptions
	defineCommand(&cli.Command{
		Name:  "ls",
		Usage: "List directory on file server.",
		Flags: defineFetchOptionsFlags(&fetchOptions, []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name",
				Destination: &name,
				Required:    true,
			},
		}),
		Before: openUplink,
		Action: func(c *cli.Context) error {
			m, e := retrieveFileMetadata(c.Context, name, fetchOptions)
			if e != nil {
				return e
			}
			if !m.IsDir() {
				return fmt.Errorf("mode %o is not a directory", m.Mode)
			}

			fetcher := segmented.Fetch(m.Name, fetchOptions)
			payload, e := fetcher.Payload(c.Context)
			if e != nil {
				return e
			}

			var ls ndn6file.DirectoryListing
			if e = ls.UnmarshalBinary(payload); e != nil {
				return e
			}

			log.Printf("retrieved %d entries in directory listing", len(ls))
			fmt.Println(ls)
			return nil
		},
	})
}

func init() {
	var name, filename string
	var fetchOptions segmented.FetchOptions
	var useTgFetcher bool
	defineCommand(&cli.Command{
		Name:  "fetch",
		Usage: "Fetch file from file server.",
		Flags: defineFetchOptionsFlags(&fetchOptions, []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "name",
				Destination: &name,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "filename",
				Usage:       "output file name",
				DefaultText: "write to stdout",
				Destination: &filename,
			},
			&cli.BoolFlag{
				Name:        "tg-fetcher",
				Usage:       "generate arguments for 'ndndpdk-ctrl start-fetch' command to fetch with NDN-DPDK service",
				Destination: &useTgFetcher,
			},
		}),
		Before: openUplink,
		Action: func(c *cli.Context) error {
			m, e := retrieveFileMetadata(c.Context, name, fetchOptions)
			if e != nil {
				return e
			}
			if !m.IsFile() {
				return fmt.Errorf("mode %o is not a file", m.Mode)
			}
			log.Printf("file has %d octets in %d segments of size %d", m.Size, m.SegmentEnd(), m.SegmentSize)
			if useTgFetcher {
				fmt.Println(shellquote.Join(
					"--name", m.Name.String(),
					"--segment-end", strconv.FormatUint(m.SegmentEnd(), 10),
					"--file-size", strconv.FormatInt(m.Size, 10),
					"--segment-len", strconv.Itoa(m.SegmentSize),
				))
				return nil
			}
			return retrieveSegmented(c.Context, m.Name, filename, m.SegmentSize, fetchOptions)
		},
	})
}
