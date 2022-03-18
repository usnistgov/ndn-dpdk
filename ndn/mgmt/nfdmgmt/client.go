// Package nfdmgmt provides access to NFD Management API.
package nfdmgmt

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

const (
	ttControlParameters = 0x68
	ttOrigin            = 0x6F
	ttFlags             = 0x6C

	ttControlResponse = 0x65
	ttStatusCode      = 0x66

	originClient = 65
	flagCapture  = 2
)

// Client provides access to NFD Management API.
type Client struct {
	ConsumerOpts endpoint.ConsumerOptions
	Prefix       string
	Signer       ndn.Signer
}

var _ mgmt.Client = (*Client)(nil)

// OpenFace creates a socket face toward NFD.
//
// Transport type and socket path are read from NDN_CLIENT_TRANSPORT environment variable.
// However, this does not support client.conf file.
// https://named-data.net/doc/ndn-cxx/0.8.0/manpages/ndn-client.conf.html
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

func (c *Client) invoke(command string, parameters ...tlv.Fielder) (status int, e error) {
	name := ndn.ParseName(c.Prefix + "/" + command)
	name = append(name, ndn.NameComponentFrom(an.TtGenericNameComponent, tlv.TLVFrom(ttControlParameters, parameters...)))
	interest := ndn.Interest{
		Name:        name,
		MustBeFresh: true,
		SigInfo: &ndn.SigInfo{
			Nonce: make([]byte, 4),
			Time:  uint64(time.Now().UnixMilli()),
		},
	}
	rand.Read(interest.SigInfo.Nonce)
	if e = c.Signer.Sign(&interest); e != nil {
		return 0, fmt.Errorf("signing error: %w", e)
	}

	data, e := endpoint.Consume(context.TODO(), interest, c.ConsumerOpts)
	if e != nil {
		return 0, fmt.Errorf("consumer error: %w", e)
	}

	d0 := tlv.DecodingBuffer(data.Content)
	var de0, de1 tlv.DecodingElement
	de0, e = d0.Element()
	if e == nil && de0.Type == ttControlResponse {
		d1 := tlv.DecodingBuffer(de0.Value)
		de1, e = d1.Element()
		if e == nil && de1.Type == ttStatusCode {
			if status = int(de1.UnmarshalNNI(999, &e, tlv.ErrRange)); e == nil {
				return status, nil
			}
		}
	}
	return 0, fmt.Errorf("decode error: %w", e)
}

// New creates a Client.
func New() (*Client, error) {
	return &Client{
		ConsumerOpts: endpoint.ConsumerOptions{
			Retx: endpoint.RetxOptions{Limit: 2},
		},
		Prefix: "/localhost/nfd",
		Signer: ndn.DigestSigning,
	}, nil
}
