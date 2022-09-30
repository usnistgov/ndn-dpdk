package nfdmgmt

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
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
	cr, e := f.client.Invoke(context.TODO(), RibRegisterCommand{
		Name:      name,
		Origin:    RouteOriginClient,
		NoInherit: true,
		Capture:   true,
	})
	if e != nil {
		return e
	}
	if cr.StatusCode != 200 {
		return fmt.Errorf("unexpected response status %d", cr.StatusCode)
	}
	return nil
}

func (f *nfdFace) Withdraw(name ndn.Name) error {
	cr, e := f.client.Invoke(context.TODO(), RibUnregisterCommand{
		Name:   name,
		Origin: RouteOriginClient,
	})
	if e != nil {
		return e
	}
	if cr.StatusCode != 200 {
		return fmt.Errorf("unexpected response status %d", cr.StatusCode)
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

	tr, e := sockettransport.Dial(uri.Scheme, "", remote, sockettransport.Config{MTU: 8800})
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
