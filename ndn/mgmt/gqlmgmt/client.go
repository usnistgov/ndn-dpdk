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

var _ mgmt.Client = (*Client)(nil)

// CreateFace requests to create a face via GraphQL.
func (c *Client) CreateFace(ctx context.Context, locator interface{}) (id string, e error) {
	var faceJ struct {
		ID string `json:"id"`
	}
	e = c.Do(ctx, `
		mutation createFace($locator: JSON!) {
			createFace(locator: $locator) {
				id
			}
		}
	`, map[string]interface{}{
		"locator": locator,
	}, "createFace", &faceJ)
	return faceJ.ID, e
}

// New creates a Client.
func New(cfg gqlclient.Config) (*Client, error) {
	c, e := gqlclient.New(cfg)
	if e != nil {
		return nil, e
	}
	return &Client{Client: c}, nil
}
