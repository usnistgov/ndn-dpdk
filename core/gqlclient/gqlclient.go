// Package gqlclient provides a GraphQL client.
package gqlclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"

	gqlws "github.com/korylprince/go-graphql-ws"
	"github.com/machinebox/graphql"
)

func parseResultData(j json.RawMessage, key string, ptr any) error {
	if key == "" {
		return json.Unmarshal([]byte(j), ptr)
	}

	m := map[string]json.RawMessage{}
	if e := json.Unmarshal([]byte(j), &m); e != nil {
		return e
	}
	return json.Unmarshal([]byte(m[key]), ptr)
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

// Do runs a query or mutation on the GraphQL server.
//  ctx: a Context for canceling the operation.
//  query: a GraphQL document.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: pointer to result struct.
func (c *Client) Do(ctx context.Context, query string, vars map[string]any, key string, res any) error {
	c.wg.Add(1)
	defer c.wg.Done()

	request := graphql.NewRequest(query)
	for key, value := range vars {
		request.Var(key, value)
	}

	var response json.RawMessage
	if e := c.httpClient.Run(ctx, request, &response); e != nil {
		var netOpError *net.OpError
		if errors.As(e, &netOpError) && netOpError.Op == "dial" {
			e = fmt.Errorf("%w; is NDN-DPDK running? is GraphQL server endpoint specified correctly?", e)
		}
		return e
	}
	return parseResultData(response, key, res)
}

// Delete runs the delete mutation.
func (c *Client) Delete(ctx context.Context, id string) (deleted bool, e error) {
	e = c.Do(ctx, `
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, map[string]any{
		"id": id,
	}, "delete", &deleted)
	return deleted, e
}

// Subscribe performs a subscription on the GraphQL server.
//  ctx: a Context for canceling the subscription.
//  query: a GraphQL document.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: channel for sending updates.
func (c *Client) Subscribe(ctx context.Context, query string, vars map[string]any, key string, res any) error {
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
	if e := cfg.Validate(); e != nil {
		return nil, e
	}

	c := &Client{
		cfg:        cfg,
		httpClient: graphql.NewClient(cfg.HTTPUri, graphql.WithHTTPClient(cfg.HTTPClient)),
		wsClosed:   make(chan struct{}),
	}
	return c, nil
}
