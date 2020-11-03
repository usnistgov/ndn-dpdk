// Package gqlclient provides a GraphQL client.
package gqlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
)

// Client is a GraphQL client.
type Client struct {
	uri        string
	HTTPClient http.Client
	wg         sync.WaitGroup
}

// Close blocks until all pending operations have concluded.
func (c *Client) Close() error {
	c.wg.Wait()
	return nil
}

// Do executes an operation on the GraphQL server.
//  query: a GraphQL document, may contain only one query or mutation.
func (c *Client) Do(query string, vars interface{}, key string, res interface{}) error {
	return c.DoOperation(query, "", vars, key, res)
}

// DoOperation executes an operation on the GraphQL server.
//  query: a GraphQL document, may contain multiple operations.
//  op: operation name.
//  vars: query variables.
//  key: if non-empty, unmarshal result.data[key] instead of result.data.
//  res: pointer to result struct.
func (c *Client) DoOperation(query, op string, vars interface{}, key string, res interface{}) error {
	c.wg.Add(1)
	defer c.wg.Done()

	params := handler.RequestOptions{
		Query:         query,
		OperationName: op,
	}
	if e := jsonhelper.Roundtrip(vars, &params.Variables); e != nil {
		return fmt.Errorf("json(vars): %w", e)
	}

	j, e := json.Marshal(params)
	if e != nil {
		return fmt.Errorf("json.Marshal(params): %w", e)
	}
	request, e := http.NewRequest(http.MethodPost, c.uri, bytes.NewReader(j))
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
	body, e := ioutil.ReadAll(response.Body)
	if e != nil {
		return fmt.Errorf("ioutil.ReadAll(response.Body): %w", e)
	}

	var result graphql.Result
	e = json.Unmarshal(body, &result)
	if e != nil {
		return fmt.Errorf("json.Unmarshal(response.Body): %w", e)
	}
	if result.HasErrors() {
		return fmt.Errorf("result.HasErrors: %w", result.Errors[0])
	}

	if res != nil {
		data := result.Data
		if key != "" {
			m := data.(map[string]interface{})
			var ok bool
			data, ok = m[key]
			if !ok {
				return fmt.Errorf("data[%s] missing", key)
			}
		}
		if e := jsonhelper.Roundtrip(data, res); e != nil {
			return fmt.Errorf("json(data): %w", e)
		}
	}
	return nil
}

// New creates a Client.
func New(uri string) (*Client, error) {
	u, e := url.Parse(uri)
	if e != nil {
		return nil, fmt.Errorf("url.Parse: %w", e)
	}

	c := &Client{
		uri: u.String(),
	}
	return c, nil
}
