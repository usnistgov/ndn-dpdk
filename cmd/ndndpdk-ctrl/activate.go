package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/xeipuuv/gojsonschema"
)

type schemaError struct {
	*gojsonschema.Result
}

func (e schemaError) Error() string {
	var b strings.Builder
	fmt.Fprintln(&b, "JSON document failed schema validation:")
	for _, desc := range e.Result.Errors() {
		fmt.Fprintln(&b, "-", desc)
	}
	return b.String()
}

func checkSchema(input gojsonschema.JSONLoader, schemaName string) error {
	exe, e := os.Executable()
	if e != nil {
		return e
	}

	schemaFile := url.URL{
		Scheme: "file",
		Path:   path.Join(path.Dir(exe), "../share/ndn-dpdk", schemaName+".schema.json"),
	}
	schema := gojsonschema.NewReferenceLoader(schemaFile.String())
	result, e := gojsonschema.Validate(schema, input)
	if e != nil {
		fmt.Fprintln(os.Stderr, "JSON schema validator error:", e)
		return e
	}

	if !result.Valid() {
		return schemaError{result}
	}
	return nil
}

func defineActivateCommand(id, noun string) {
	var skipSchema bool
	defineCommand(&cli.Command{
		Category: "activate",
		Name:     "activate-" + id,
		Usage:    "Activate ndndpdk-svc as " + noun + " (pass config via stdin)",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "skip-schema",
				Usage:       "do not check JSON schema",
				Value:       false,
				Destination: &skipSchema,
			},
		},
		Action: func(c *cli.Context) error {
			arg := make(map[string]interface{})
			loader, stdin := gojsonschema.NewReaderLoader(os.Stdin)
			decoder := json.NewDecoder(stdin)

			hasInput := make(chan bool, 1)
			go func() {
				delay := time.NewTimer(2 * time.Second)
				defer delay.Stop()
				select {
				case <-hasInput:
				case <-delay.C:
					fmt.Fprintln(os.Stderr, "Hint: pass config via stdin")
				}
			}()

			e := decoder.Decode(&arg)
			hasInput <- true
			if e != nil {
				return e
			}

			if !skipSchema {
				if e := checkSchema(loader, id); e != nil {
					return e
				}
			}
			return clientDoPrint(fmt.Sprintf(`
				mutation activate($arg: JSON!) {
					activate(%s: $arg)
				}
			`, id), map[string]interface{}{
				"arg": arg,
			}, "activate")
		},
	})
}

func init() {
	defineActivateCommand("forwarder", "forwarder")
	defineActivateCommand("trafficgen", "traffic generator")

	defineCommand(&cli.Command{
		Category: "activate",
		Name:     "shutdown",
		Usage:    "Shutdown NDN-DPDK service",
		Action: func(c *cli.Context) error {
			return clientDoPrint(`
				mutation shutdown {
					shutdown
				}
			`, nil, "shutdown")
		},
	})
}
