package mgmt

import (
	"net"
	"net/rpc"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/powerman/rpc-codec/jsonrpc2"
)

var (
	server   = rpc.NewServer()
	listener net.Listener
)

// Register registers a management module.
// Errors are fatal.
func Register(mg interface{}) {
	typeName := reflect.TypeOf(mg).Name()
	name := strings.TrimSuffix(typeName, "Mgmt")

	logEntry := log.WithField("mg", name)
	if e := server.RegisterName(name, mg); e != nil {
		logEntry.WithError(e).Fatal("register failed")
	}
	logEntry.Debug("registered")
}

// Start starts the management listener.
// Errors are fatal.
func Start() {
	if listener != nil {
		log.Fatal("listener already started")
	}

	mgmtEnv := os.Getenv("MGMT")
	if mgmtEnv == "0" {
		return
	}

	if mgmtEnv == "" {
		mgmtEnv = "unix:///var/run/ndn-dpdk-mgmt.sock"
	}

	u, e := url.Parse(mgmtEnv)
	if e != nil {
		log.WithError(e).Fatal("cannot parse MGMT environment variable")
	}

	var addr string
	switch u.Scheme {
	case "unix":
		addr = u.Path
		os.Remove(addr)
	case "tcp", "tcp4", "tcp6":
		addr = u.Host
	default:
		log.Fatalf("unsupported MGMT scheme %s", u.Scheme)
	}

	listener, e = net.Listen(u.Scheme, addr)
	if e != nil {
		log.Fatalf("cannot listen on %s %s", u.Scheme, addr)
	}

	go serve()
}

func serve() {
	for {
		conn, e := listener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				continue
			} else {
				break
			}
		}
		go server.ServeCodec(jsonrpc2.NewServerCodec(conn, server))
	}
}
