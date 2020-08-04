// Package gqlmgmt provides access to NDN-DPDK GraphQL API.
package gqlmgmt

import (
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
)

// Client provides access to NDN-DPDK GraphQL API.
type Client struct {
	*gqlclient.Client
}

var _ mgmt.Client = (*Client)(nil)

// New creates a Client.
func New(uri string) (*Client, error) {
	c, e := gqlclient.New(uri)
	if e != nil {
		return nil, e
	}
	return &Client{Client: c}, nil
}
