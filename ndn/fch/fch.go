// Package fch provides a simple NDN-FCH client.
// https://github.com/11th-ndn-hackathon/ndn-fch
package fch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/schema"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/zyedidia/generic"
)

var encoder = schema.NewEncoder()

const (
	DefaultServer    = "https://fch.ndn.today"
	DefaultTransport = "udp"
)

// Request represents an NDN-FCH request.
type Request struct {
	// Server is the base URI of the NDN- server.
	Server string `schema:"-"`

	// Transport specifies a transport protocol.
	Transport string `schema:"cap"`

	// Count specifies number of requested routers.
	Count int `schema:"k"`
}

func (req *Request) applyDefaults() {
	if req.Server == "" {
		req.Server = DefaultServer
	}
	req.Count = generic.Max(1, req.Count)
	if req.Transport == "" {
		req.Transport = DefaultTransport
	}
}

func (req Request) toURL() (u *url.URL, e error) {
	if u, e = url.Parse(req.Server); e != nil {
		return nil, e
	}
	qs := url.Values{}
	if e = encoder.Encode(req, qs); e != nil {
		return nil, e
	}
	u.RawQuery = qs.Encode()
	return u, nil
}

// Response represents an NDN-FCH response.
type Response struct {
	Updated uint64   `json:"updated"`
	Routers []Router `json:"routers"`
}

// Router describes a router in NDN-FCH response.
type Router struct {
	Transport string   `json:"transport"`
	Connect   string   `json:"connect"`
	Prefix    ndn.Name `json:"prefix,omitempty"`
}

// Query performs an NDN-FCH query.
func Query(ctx context.Context, req Request) (res Response, e error) {
	req.applyDefaults()
	u, e := req.toURL()
	if e != nil {
		return res, e
	}

	hReq, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if e != nil {
		return res, e
	}
	hReq.Header.Set("Accept", "application/json, text/plain, */*")

	hRes, e := http.DefaultClient.Do(hReq)
	if e != nil {
		return res, e
	}
	if hRes.StatusCode != http.StatusOK {
		return res, fmt.Errorf("HTTP %s", hRes.Status)
	}

	body, e := io.ReadAll(hRes.Body)
	if e != nil {
		return res, e
	}

	if strings.HasPrefix(hRes.Header.Get("Content-Type"), "application/json") {
		e = json.Unmarshal(body, &res)
		return res, e
	}

	routers := bytes.Split(body, []byte{','})
	for _, router := range routers {
		if len(router) == 0 {
			return res, errors.New("empty response")
		}

		connect := string(router)
		if req.Transport == "udp" {
			if _, _, e := net.SplitHostPort(connect); e != nil {
				connect = net.JoinHostPort(connect, "6363")
			}
		}

		res.Routers = append(res.Routers, Router{
			Transport: req.Transport,
			Connect:   connect,
		})
	}
	return res, nil
}
