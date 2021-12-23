package nfdmgmt

import (
	"fmt"
	"net/url"
	"os"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type nfdFace struct {
	client *Client
	l3face l3.Face
}

func (f *nfdFace) ID() string {
	return ""
}

func (f *nfdFace) Face() l3.Face {
	return f.l3face
}

func (f *nfdFace) Close() error {
	return nil
}

func (f *nfdFace) Advertise(name ndn.Name) error {
	status, e := f.client.invoke("rib/register",
		name,
		tlv.TLVNNI(ttOrigin, originClient),
		tlv.TLVNNI(ttFlags, flagCapture),
	)
	if e != nil {
		return e
	}
	if status != 200 {
		return fmt.Errorf("unexpected response status %d", status)
	}
	return nil
}

func (f *nfdFace) Withdraw(name ndn.Name) error {
	status, e := f.client.invoke("rib/unregister",
		name,
		tlv.TLVNNI(ttOrigin, originClient),
	)
	if e != nil {
		return e
	}
	if status != 200 {
		return fmt.Errorf("unexpected response status %d", status)
	}
	return nil
}

func newNfdFace(c *Client) (f *nfdFace, e error) {
	env := os.Getenv("NDN_CLIENT_TRANSPORT")
	if env == "" {
		env = "unix:///run/nfd.sock"
	}

	uri, e := url.Parse(env)
	if e != nil {
		return nil, fmt.Errorf("bad NDN_CLIENT_TRANSPORT: %w", e)
	}

	remote := uri.Host
	if remote == "" {
		remote = uri.Path
	}

	tr, e := sockettransport.Dial(uri.Scheme, "", remote)
	if e != nil {
		return nil, e
	}

	l3face, e := l3.NewFace(tr, l3.FaceConfig{})
	if e != nil {
		return nil, e
	}

	return &nfdFace{
		client: c,
		l3face: l3face,
	}, nil
}
