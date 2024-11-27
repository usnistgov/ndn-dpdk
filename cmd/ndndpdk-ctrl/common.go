package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/xeipuuv/gojsonschema"
)

func waitInterrupt() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	defer signal.Stop(interrupt)
	<-interrupt
}

func runDeleteCommand(c *cli.Context, id string) error {
	return clientDoPrint(c.Context, `
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, map[string]any{
		"id": id,
	}, "delete")
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
			return runDeleteCommand(c, id)
		},
	})
}

type schemaError struct {
	*gojsonschema.Result
	SchemaURL *url.URL
}

func (e schemaError) Error() string {
	b := fmt.Appendln(nil, "JSON document failed schema validation:")
	for _, desc := range e.Result.Errors() {
		b = fmt.Appendln(b, "-", desc)
	}
	b = fmt.Appendln(b, "Schema", e.SchemaURL)
	return string(b)
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
	ParamNoun  string
	Flags      []cli.Flag
	Action     func(c *cli.Context, arg map[string]any) error
}

func defineStdinJSONCommand(opts stdinJSONCommand) {
	if opts.ParamNoun == "" {
		opts.ParamNoun = "parameters"
	}
	var skipSchema bool
	defineCommand(&cli.Command{
		Category: opts.Category,
		Name:     opts.Name,
		Usage:    opts.Usage + " (pass " + opts.ParamNoun + " via stdin)",
		Flags: append([]cli.Flag{
			&cli.BoolFlag{
				Name:        "skip-schema",
				Usage:       "do not check JSON schema",
				Value:       false,
				Destination: &skipSchema,
			},
		}, opts.Flags...),
		Action: func(c *cli.Context) error {
			arg := map[string]any{}
			loader, stdin := gojsonschema.NewReaderLoader(os.Stdin)
			decoder := json.NewDecoder(stdin)

			hasInput := make(chan struct{})
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
			close(hasInput)
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

type request struct {
	Query string
	Vars  map[string]any
	Key   string
}

func (r request) isSubscription() bool {
	var verb string
	_, e := fmt.Sscan(r.Query, &verb)
	return e == nil && verb == "subscription"
}

func (r request) Execute(ctx context.Context, ptr any) error {
	if r.isSubscription() {
		return r.subscribe(ctx, reflect.ValueOf(ptr))
	}
	return r.do(ctx, ptr)
}

func (r request) do(ctx context.Context, ptr any) error {
	var value any
	if e := client.Do(ctx, r.Query, r.Vars, r.Key, &value); e != nil {
		return e
	}

	if ptr != nil {
		jsonhelper.Roundtrip(value, ptr)
	}

	if val := reflect.ValueOf(value); val.Kind() == reflect.Slice {
		for i := range val.Len() {
			j, _ := json.Marshal(val.Index(i).Interface())
			fmt.Println(string(j))
		}
	} else {
		j, _ := json.Marshal(value)
		fmt.Println(string(j))
	}
	return nil
}

func (r request) subscribe(ctx context.Context, cb reflect.Value) error {
	var elemTyp reflect.Type
	switch cb.Kind() {
	case reflect.Invalid:
	case reflect.Func:
		if cb.IsNil() {
			break
		}
		if t := cb.Type(); t.NumIn() == 1 && (t.NumOut() == 0 || (t.NumOut() == 1 && t.Out(0).Kind() == reflect.Bool)) {
			elemTyp = t.In(0)
			break
		}
		fallthrough
	default:
		panic("invalid callback")
	}

	subscribeCtx, unsubscribe := context.WithCancel(ctx)
	defer unsubscribe()

	updates := make(chan any)
	go func() {
		for update := range updates {
			j, _ := json.Marshal(update)
			fmt.Println(string(j))

			if elemTyp != nil {
				elemPtr := reflect.New(elemTyp)
				if e := json.Unmarshal(j, elemPtr.Interface()); e == nil {
					ret := cb.Call([]reflect.Value{elemPtr.Elem()})
					if len(ret) == 1 && !ret[0].Bool() {
						unsubscribe()
					}
				}
			}
		}
	}()
	return client.Subscribe(subscribeCtx, r.Query, r.Vars, r.Key, updates)
}

func (r request) Print() error {
	query := []byte(r.Query)
	if bytes.HasPrefix(query, []byte("\n\t")) {
		prefixLen := len(query) - len(bytes.TrimLeft(query[1:], "\t"))
		query = bytes.ReplaceAll(query, query[:prefixLen], []byte("\n\t"))
	}
	query = bytes.TrimRight(query, "\n\t")
	query = bytes.ReplaceAll(query, []byte("\t"), []byte("  "))
	query = append(query, '\n')

	gqArgs := []string{gqlCfg.HTTPUri, "-q", string(query)}
	if r.isSubscription() {
		gqArgs[0] = strings.Replace(gqlCfg.WebSocketUri, "ws", "http", 1)
	}
	if r.Vars != nil {
		j, e := json.MarshalIndent(r.Vars, "", "  ")
		if e != nil {
			return e
		}
		gqArgs = append(gqArgs, "--variablesJSON", string(j))
	}

	jqArgs := []string{"-c"}
	if r.Key == "" {
		jqArgs = append(jqArgs, ".data")
	} else {
		jqArgs = append(jqArgs, ".data."+r.Key)
	}

	fmt.Println("corepack", "pnpm", "-s", "dlx", "graphqurl", shellquote.Join(gqArgs...), "|", "jq", shellquote.Join(jqArgs...))
	fmt.Println()
	return nil
}

// clientDoPrint either runs a GraphQL query/mutation/subscription or prints the command to stdout.
//
//	query: GraphQL document, which should contain exactly one GraphQL operation.
//	vars: query variables.
//	key: top-level JSON property key to extract from response.
//	ptr: for query/mutation, pointer to a variable for receiving the response;
//	     for subscription, `func(T)` or `func(T) bool` for receiving each update, return false to unsubscribe.
func clientDoPrint(ctx context.Context, query string, vars map[string]any, key string, ptr ...any) error {
	r := request{
		Query: query,
		Vars:  vars,
		Key:   key,
	}

	if cmdout {
		return r.Print()
	}
	if len(ptr) == 0 {
		return r.Execute(ctx, nil)
	}
	return r.Execute(ctx, ptr[0])
}
