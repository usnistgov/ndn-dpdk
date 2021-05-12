// Package gqlclient provides a GraphQL client.
package gqlclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"sync"

	gqlws "github.com/korylprince/go-graphql-ws"
	"github.com/machinebox/graphql"
)

func parseResultData(j json.RawMessage, key string, ptr interface{}) error {
	if key == "" {
		return json.Unmarshal([]byte(j), ptr)
	}

	m := make(map[string]json.RawMessage)
	if e := json.Unmarshal([]byte(j), &m); e != nil {
		return e
	}
	return json.Unmarshal([]byte(m[key]), ptr)
}

// Config contains Client configuration.
type Config struct {
	// HTTPUri is HTTP URI for query and mutation operations.
	HTTPUri string

	HTTPClient *http.Client

	// WebSocketUri is WebSocket URI for subscription operations.
	// Default is appending '/subscriptions' to HTTPUri.
	WebSocketUri string

	WebSocketDialer *gqlws.Dialer
}

// ApplyDefaults applies defaults.
func (cfg *Config) ApplyDefaults() error {
	u, e := url.Parse(cfg.HTTPUri)
	if e != nil {
		return fmt.Errorf("HTTPUri: %w", e)
	}
	cfg.HTTPUri = u.String()

	if cfg.WebSocketUri == "" {
		switch u.Scheme {
		case "http":
			u.Scheme = "ws"
		case "https":
			u.Scheme = "wss"
		}
		u.Path = path.Join(u.Path, "subscriptions")
		cfg.WebSocketUri = u.String()
	} else {
		u, e = url.Parse(cfg.WebSocketUri)
		if e != nil {
			return fmt.Errorf("WebSocketUri: %w", e)
		}
		cfg.WebSocketUri = u.String()
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	if cfg.WebSocketDialer == nil {
		dialer := *gqlws.DefaultDialer
		dialer.Subprotocols = []string{"graphql-ws"}
		cfg.WebSocketDialer = &dialer
	}

	return nil
}

// Client is a GraphQL client.
type Client struct {
	cfg        Config
	wg         sync.WaitGroup
	httpClient *graphql.Client

	wsConnMutex sync.Mutex
	wsConn      *gqlws.Conn
	wsConnErr   error
	wsClosed    chan struct{}
}

// Close blocks until all pending operations have concluded.
func (c *Client) Close() error {
	c.wg.Wait()

	c.wsConnMutex.Lock()
	defer c.wsConnMutex.Unlock()
	if c.wsConn != nil {
		c.wsConn.Close()
	}

	return nil
}

// Do executes a query or mutation on the GraphQL server.
//  ctx: a Context for canceling the operation.
//  query: a GraphQL document.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: pointer to result struct.
func (c *Client) Do(ctx context.Context, query string, vars map[string]interface{}, key string, res interface{}) error {
	c.wg.Add(1)
	defer c.wg.Done()

	request := graphql.NewRequest(query)
	for key, value := range vars {
		request.Var(key, value)
	}

	var response json.RawMessage
	if e := c.httpClient.Run(ctx, request, &response); e != nil {
		return e
	}
	return parseResultData(response, key, res)
}

// Subscribe executes a subscription on the GraphQL server.
//  ctx: a Context for canceling the subscription.
//  query: a GraphQL document.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: channel for sending updates.
func (c *Client) Subscribe(ctx context.Context, query string, vars map[string]interface{}, key string, res interface{}) error {
	c.wg.Add(1)
	defer c.wg.Done()

	resR := reflect.ValueOf(res)
	defer resR.Close()
	valueTyp := resR.Type().Elem()

	conn, e := c.wsConnect()
	if e != nil {
		return e
	}

	fail := make(chan error, 1)

	id, e := conn.Subscribe(&gqlws.MessagePayloadStart{
		Query:     query,
		Variables: vars,
	}, func(message *gqlws.Message) {
		switch message.Type {
		case gqlws.MessageTypeError:
			fail <- gqlws.ParseError(message.Payload)

		case gqlws.MessageTypeComplete:
			fail <- nil

		case gqlws.MessageTypeData:
			var payload gqlws.MessagePayloadData
			if e := json.Unmarshal([]byte(message.Payload), &payload); e != nil {
				fail <- e
				return
			}

			if len(payload.Errors) > 0 {
				fail <- payload.Errors
				return
			}

			value := reflect.New(valueTyp)
			if e := parseResultData(payload.Data, key, value.Interface()); e != nil {
				fail <- e
				return
			}

			resR.Send(value.Elem())
		}
	})
	if e != nil {
		return e
	}
	defer conn.Unsubscribe(id)

	select {
	case <-ctx.Done():
		return nil
	case <-c.wsClosed:
		return io.ErrUnexpectedEOF
	case e := <-fail:
		return e
	}
}

func (c *Client) wsConnect() (conn *gqlws.Conn, e error) {
	c.wsConnMutex.Lock()
	defer c.wsConnMutex.Unlock()

	if c.wsConn == nil && c.wsConnErr == nil {
		c.wg.Add(1)
		defer c.wg.Done()
		c.wsConn, _, c.wsConnErr = c.cfg.WebSocketDialer.Dial(c.cfg.WebSocketUri, nil, &gqlws.MessagePayloadConnectionInit{})
		if c.wsConnErr == nil {
			c.wsConn.SetCloseHandler(func(int, string) {
				close(c.wsClosed)
			})
		}
	}
	return c.wsConn, c.wsConnErr
}

// New creates a Client.
func New(cfg Config) (*Client, error) {
	if e := cfg.ApplyDefaults(); e != nil {
		return nil, e
	}

	c := &Client{
		cfg:        cfg,
		httpClient: graphql.NewClient(cfg.HTTPUri, graphql.WithHTTPClient(cfg.HTTPClient)),
		wsClosed:   make(chan struct{}),
	}
	return c, nil
}
