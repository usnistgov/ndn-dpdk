// Package gqlmgmt provides access to NDN-DPDK GraphQL API.
package gqlmgmt

import (
	"context"

	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
)

// Client provides access to NDN-DPDK GraphQL API.
type Client struct {
	*gqlclient.Client
}

func (c *Client) delete(id string) error {
	deleted := false
	return c.Do(context.TODO(), `
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, map[string]interface{}{
		"id": id,
	}, "delete", &deleted)
}

var _ mgmt.Client = (*Client)(nil)

// New creates a Client.
func New(cfg gqlclient.Config) (*Client, error) {
	c, e := gqlclient.New(cfg)
	if e != nil {
		return nil, e
	}
	return &Client{Client: c}, nil
}
