package gqlclient

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	gqlws "github.com/korylprince/go-graphql-ws"
)

// Config contains Client configuration.
type Config struct {
	// HTTPUri is HTTP URI for query and mutation operations.
	HTTPUri string

	HTTPClient *http.Client

	// WebSocketUri is WebSocket URI for subscription operations.
	// Default is same as HTTPURI except changing the scheme.
	WebSocketUri string

	WebSocketDialer *gqlws.Dialer
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	u, e := url.Parse(cfg.HTTPUri)
	if e != nil {
		return fmt.Errorf("HTTPUri: %w", e)
	}
	cfg.HTTPUri = u.String()

	if cfg.WebSocketUri == "" {
		switch u.Scheme {
		case "http":
			u.Scheme = "ws"
		case "https":
			u.Scheme = "wss"
		}
		cfg.WebSocketUri = u.String()
	} else {
		u, e = url.Parse(cfg.WebSocketUri)
		if e != nil {
			return fmt.Errorf("WebSocketUri: %w", e)
		}
		cfg.WebSocketUri = u.String()
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	if cfg.WebSocketDialer == nil {
		dialer := *gqlws.DefaultDialer
		dialer.Subprotocols = []string{"graphql-ws"}
		cfg.WebSocketDialer = &dialer
	}

	return nil
}

// Listen constructs server listen string.
func (cfg Config) Listen() (hostport string, e error) {
	uri, e := url.Parse(cfg.HTTPUri)
	if e != nil {
		return "", e
	}

	if uri.Scheme != "http" || strings.TrimPrefix(uri.Path, "/") != "" {
		return "", errors.New("GraphQL server URI should have 'http' scheme and '/' path")
	}

	host, port := uri.Hostname(), uri.Port()
	if port == "" {
		port = "80"
	}
	return net.JoinHostPort(host, port), nil
}

// MakeListenAddress constructs server listen string.
func MakeListenAddress(uri string) (hostport string, e error) {
	cfg := Config{HTTPUri: uri}
	return cfg.Listen()
}
