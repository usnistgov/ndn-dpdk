// Package gqlclient provides a GraphQL client.
package gqlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"sync"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/functionalfoundry/graphqlws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"go.uber.org/multierr"
)

func encodeParams(query, op string, vars interface{}, wsID string) ([]byte, error) {
	var message interface{}
	var varsMap *map[string]interface{}

	if wsID == "" {
		params := handler.RequestOptions{
			Query:         query,
			OperationName: op,
		}
		varsMap = &params.Variables
		message = params
	} else {
		payload := graphqlws.StartMessagePayload{
			Query:         query,
			OperationName: op,
		}
		varsMap = &payload.Variables
		message = graphqlws.OperationMessage{
			ID:      wsID,
			Type:    "start",
			Payload: &payload,
		}
	}

	if e := jsonhelper.Roundtrip(vars, varsMap); e != nil {
		return nil, fmt.Errorf("json(vars): %w", e)
	}

	j, e := json.Marshal(message)
	if e != nil {
		return nil, fmt.Errorf("json.Marshal(message): %w", e)
	}
	return j, nil
}

func decodeResult(data interface{}, key string, res interface{}) error {
	if key != "" {
		m, ok := data.(map[string]interface{})
		if !ok {
			return errors.New("data is not an object")
		}

		data, ok = m[key]
		if !ok {
			return fmt.Errorf("data[%s] missing", key)
		}
	}

	if e := jsonhelper.Roundtrip(data, res); e != nil {
		return fmt.Errorf("json(data): %w", e)
	}
	return nil
}

// Client is a GraphQL client.
type Client struct {
	httpUri    string
	wsUri      string
	HTTPClient http.Client
	wg         sync.WaitGroup
}

// Close blocks until all pending operations have concluded.
func (c *Client) Close() error {
	c.wg.Wait()
	return nil
}

// Do executes a query or mutation on the GraphQL server.
//  ctx: a Context for canceling the operation.
//  query: a GraphQL document.
//  op: operation name; may be empty if the GraphQL document has only one operation.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: pointer to result struct.
func (c *Client) Do(ctx context.Context, query, op string, vars interface{}, key string, res interface{}) error {
	c.wg.Add(1)
	defer c.wg.Done()

	j, e := encodeParams(query, op, vars, "")
	if e != nil {
		return e
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodPost, c.httpUri, bytes.NewReader(j))
	if e != nil {
		return fmt.Errorf("http.NewRequest: %w", e)
	}
	request.Header.Add("accept", "application/json")

	response, e := c.HTTPClient.Do(request)
	if e != nil {
		return fmt.Errorf("http.Post: %w", e)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("response.Status: %s", response.Status)
	}
	body, e := io.ReadAll(response.Body)
	if e != nil {
		return fmt.Errorf("io.ReadAll(response.Body): %w", e)
	}

	var result graphql.Result
	e = json.Unmarshal(body, &result)
	if e != nil {
		return fmt.Errorf("json.Unmarshal(response.Body): %w", e)
	}
	if result.HasErrors() {
		errs := make([]error, len(result.Errors))
		for i, e := range result.Errors {
			errs[i] = e
		}
		return fmt.Errorf("result.HasErrors: %w", multierr.Combine(errs...))
	}

	if res != nil {
		if e := decodeResult(result.Data, key, res); e != nil {
			return e
		}
	}
	return nil
}

// Subscribe executes a subscription on the GraphQL server.
//  ctx: a Context for canceling the subscription.
//  query: a GraphQL document.
//  op: operation name; may be empty if the GraphQL document has only one operation.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: channel for sending updates.
func (c *Client) Subscribe(ctx context.Context, query, op string, vars interface{}, key string, res interface{}) error {
	j, e := encodeParams(query, op, vars, "w")
	if e != nil {
		return e
	}

	conn, response, e := websocket.DefaultDialer.DialContext(ctx, c.wsUri, http.Header{"Sec-WebSocket-Protocol": []string{"graphql-ws"}})
	if e != nil {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("websocket.Dial: %w\n%s", e, string(body))
	}
	defer conn.Close()

	if e := conn.WriteMessage(websocket.TextMessage, j); e != nil {
		return fmt.Errorf("conn.WriteMessage(start): %w", e)
	}

	fail := make(chan error, 2)
	messages := make(chan []byte)
	go func() {
		defer close(messages)
		for {
			_, message, e := conn.ReadMessage()
			if e != nil {
				fail <- e
				break
			}
			messages <- message
		}
	}()
	go func() {
		resR := reflect.ValueOf(res)
		defer resR.Close()
		valueTyp := resR.Type().Elem()
		for j := range messages {
			var message graphqlws.OperationMessage
			if e := json.Unmarshal(j, &message); e != nil {
				fail <- fmt.Errorf("json.Unmarshal(message): %w", e)
				break
			}
			if message.Type != "data" {
				continue
			}

			var payload graphqlws.DataMessagePayload
			if e := jsonhelper.Roundtrip(message.Payload, &payload); e != nil {
				fail <- fmt.Errorf("json(message): %w", e)
				break
			}

			if len(payload.Errors) > 0 {
				fail <- multierr.Combine(payload.Errors...)
				break
			}

			value := reflect.New(valueTyp)
			if e := decodeResult(payload.Data, key, value.Interface()); e != nil {
				fail <- e
				break
			}

			resR.Send(value.Elem())
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case e := <-fail:
		return e
	}
}

// New creates a Client.
func New(uri string) (*Client, error) {
	httpUri, e := url.Parse(uri)
	if e != nil {
		return nil, fmt.Errorf("url.Parse: %w", e)
	}

	wsUri := *httpUri
	switch wsUri.Scheme {
	case "http":
		wsUri.Scheme = "ws"
	case "https":
		wsUri.Scheme = "wss"
	}
	wsUri.Path = path.Join(wsUri.Path, "subscriptions")

	c := &Client{
		httpUri: httpUri.String(),
		wsUri:   wsUri.String(),
	}
	return c, nil
}
