// Package nfdmgmt provides access to NFD Management API.
package nfdmgmt

import (
	"context"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Client provides access to NFD Management API.
type Client struct {
	ConsumerOpts endpoint.ConsumerOptions
	Prefix       ndn.Name
	Signer       ndn.Signer
}

var _ mgmt.Client = (*Client)(nil)

// OpenFace creates a socket face toward NFD.
//
// Transport type and socket path are read from NDN_CLIENT_TRANSPORT environment variable.
// However, this does not support client.conf file.
// https://docs.named-data.net/ndn-cxx/0.8.1/manpages/ndn-client.conf.html
//
// NFD uses in-band management where the prefix registration commands are sent over the same face.
// To use prefix registration feature, calling code need to:
// 1. Add the l3.Face to an l3.Forwarder, and set the l3.Forwarder into c.ConsumerOpts if it's not the default.
// 2. Add a route toward the face for c.Prefix (default is /localhost/nfd).
// 3. Set c.Signer to a key accepted by NFD (default is digest signing).
func (c *Client) OpenFace() (mgmt.Face, error) {
	f, e := newNfdFace(c)
	return f, e
}

// Close does nothing.
func (c *Client) Close() error {
	return nil
}

// Invoke invokes a control command.
func (c *Client) Invoke(ctx context.Context, cmd ControlCommand) (cr ControlResponse, e error) {
	interest := MakeCommandInterest(c.Prefix, cmd)
	if e = c.Signer.Sign(&interest); e != nil {
		return cr, fmt.Errorf("signing error: %w", e)
	}

	data, e := endpoint.Consume(ctx, interest, c.ConsumerOpts)
	if e != nil {
		return cr, fmt.Errorf("consumer error: %w", e)
	}

	e = tlv.Decode(data.Content, &cr)
	return cr, e
}

// New creates a Client.
func New() (*Client, error) {
	return &Client{
		ConsumerOpts: endpoint.ConsumerOptions{
			Retx: endpoint.RetxOptions{Limit: 2},
		},
		Prefix: PrefixLocalhost,
		Signer: ndn.DigestSigning,
	}, nil
}
