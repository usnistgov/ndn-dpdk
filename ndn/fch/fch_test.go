package fch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/fch"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

var (
	makeAR    = testenv.MakeAR
	nameEqual = ndntestenv.NameEqual
)

func TestJSON(t *testing.T) {
	assert, require := makeAR(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log(r.URL)
		q := r.URL.Query()
		assert.Equal([]string{"udp"}, q["cap"])
		assert.Equal([]string{"1"}, q["k"])
		assert.False(q.Has("network"))

		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(`
			{
				"updated": 1677251440666,
				"routers": [
					{
						"transport": "udp",
						"connect": "192.0.2.4:16363",
						"prefix": "/example/router/prefix"
					}
				]
			}
		`))
	}))
	defer server.Close()

	res, e := fch.Query(context.Background(), fch.Request{
		Server: server.URL,
	})
	require.NoError(e)
	assert.Equal(int64(1677251440666), res.UpdatedTime().UnixMilli())
	require.Len(res.Routers, 1)
	assert.Equal("udp", res.Routers[0].Transport)
	assert.Equal("192.0.2.4:16363", res.Routers[0].Connect)
	nameEqual(assert, "/example/router/prefix", res.Routers[0].Prefix)
}

func TestTextUDP(t *testing.T) {
	assert, require := makeAR(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log(r.URL)
		q := r.URL.Query()
		assert.Equal([]string{"udp"}, q["cap"])
		assert.Equal([]string{"4"}, q["k"])
		assert.Equal([]string{"demo"}, q["network"])

		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte(`router0.example.net,router1.example.net:16363`))
	}))
	defer server.Close()

	res, e := fch.Query(context.Background(), fch.Request{
		Server:    server.URL,
		Transport: "udp",
		Count:     4,
		Network:   "demo",
	})
	require.NoError(e)
	assert.Zero(res.UpdatedTime())
	require.Len(res.Routers, 2)
	assert.Equal("udp", res.Routers[0].Transport)
	assert.Equal("router0.example.net:6363", res.Routers[0].Connect)
	assert.Nil(res.Routers[0].Prefix)
	assert.Equal("udp", res.Routers[1].Transport)
	assert.Equal("router1.example.net:16363", res.Routers[1].Connect)
	assert.Nil(res.Routers[1].Prefix)
}

func TestTextWebSocket(t *testing.T) {
	assert, require := makeAR(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log(r.URL)
		q := r.URL.Query()
		assert.Equal([]string{"wss"}, q["cap"])
		assert.Equal([]string{"2"}, q["k"])
		assert.False(q.Has("network"))

		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte(`router0.example.net,wss://router1.example.net/ndn/`))
	}))
	defer server.Close()

	res, e := fch.Query(context.Background(), fch.Request{
		Server:    server.URL,
		Transport: "wss",
		Count:     2,
	})
	require.NoError(e)
	assert.Zero(res.UpdatedTime())
	require.Len(res.Routers, 2)
	assert.Equal("wss", res.Routers[0].Transport)
	assert.Equal("wss://router0.example.net/ws/", res.Routers[0].Connect)
	assert.Nil(res.Routers[0].Prefix)
	assert.Equal("wss", res.Routers[1].Transport)
	assert.Equal("wss://router1.example.net/ndn/", res.Routers[1].Connect)
	assert.Nil(res.Routers[1].Prefix)
}
