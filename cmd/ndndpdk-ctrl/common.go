package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
	"github.com/xeipuuv/gojsonschema"
)

func defineCommand(command *cli.Command) {
	app.Commands = append(app.Commands, command)
}

func defineDeleteCommand(category, commandName, usage, objectNoun string) {
	var id string
	defineCommand(&cli.Command{
		Category: category,
		Name:     commandName,
		Usage:    usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "id",
				Usage:       objectNoun + " `ID`",
				Destination: &id,
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			return clientDoPrint(c.Context, `
				mutation delete($id: ID!) {
					delete(id: $id)
				}
			`, map[string]interface{}{
				"id": id,
			}, "delete")
		},
	})
}

type schemaError struct {
	*gojsonschema.Result
	SchemaURL *url.URL
}

func (e schemaError) Error() string {
	var b strings.Builder
	fmt.Fprintln(&b, "JSON document failed schema validation:")
	for _, desc := range e.Result.Errors() {
		fmt.Fprintln(&b, "-", desc)
	}
	fmt.Fprintln(&b, "Schema", e.SchemaURL)
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
		return schemaError{Result: result, SchemaURL: &schemaFile}
	}
	return nil
}

type stdinJSONCommand struct {
	Category   string
	Name       string
	Usage      string
	SchemaName string
	Flags      []cli.Flag
	Action     func(c *cli.Context, arg map[string]interface{}) error
}

func defineStdinJSONCommand(opts stdinJSONCommand) {
	var skipSchema bool
	defineCommand(&cli.Command{
		Category: opts.Category,
		Name:     opts.Name,
		Usage:    opts.Usage + " (pass parameters via stdin)",
		Flags: append([]cli.Flag{
			&cli.BoolFlag{
				Name:        "skip-schema",
				Usage:       "do not check JSON schema",
				Value:       false,
				Destination: &skipSchema,
			},
		}, opts.Flags...),
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
					fmt.Fprintln(os.Stderr, "Hint: pass parameters via stdin")
				}
			}()

			e := decoder.Decode(&arg)
			hasInput <- true
			if e != nil {
				return e
			}

			if !skipSchema {
				if e := checkSchema(loader, opts.SchemaName); e != nil {
					return e
				}
			}
			return opts.Action(c, arg)
		},
	})
}

func clientDoPrint(ctx context.Context, query string, vars map[string]interface{}, key string) error {
	if cmdout {
		gqArgs := []string{gqlserver, "-q", query}
		if vars != nil {
			j, e := json.MarshalIndent(vars, "", "  ")
			if e != nil {
				return e
			}
			gqArgs = append(gqArgs, "--variablesJSON", string(j))
		}
		jqArgs := []string{"-c"}
		if key == "" {
			jqArgs = append(jqArgs, ".data")
		} else {
			jqArgs = append(jqArgs, ".data."+key)
		}
		fmt.Println("gq", shellquote.Join(gqArgs...), "|", "jq", shellquote.Join(jqArgs...))
		return nil
	}

	var value interface{}
	e := client.Do(ctx, query, vars, key, &value)
	if e != nil {
		return e
	}

	if val := reflect.ValueOf(value); val.Kind() == reflect.Slice {
		for i, last := 0, val.Len(); i < last; i++ {
			j, _ := json.Marshal(val.Index(i).Interface())
			fmt.Println(string(j))
		}
	} else {
		j, _ := json.Marshal(value)
		fmt.Println(string(j))
	}
	return nil
}
