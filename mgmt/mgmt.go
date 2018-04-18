package mgmt

import (
	"fmt"
	"net"
	"net/rpc"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/powerman/rpc-codec/jsonrpc2"
)

var Server = rpc.NewServer()

func Register(mg interface{}) error {
	typeName := reflect.TypeOf(mg).Name()
	name := strings.TrimSuffix(typeName, "Mgmt")
	return Server.RegisterName(name, mg)
}

var listener net.Listener
var isClosing bool

func Start() error {
	if listener != nil {
		return fmt.Errorf("already started")
	}

	mgmtEnv := os.Getenv("MGMT")
	if mgmtEnv == "0" {
		return nil
	}

	if mgmtEnv == "" {
		mgmtEnv = "unix:///var/run/ndn-dpdk-mgmt.sock"
	}

	u, e := url.Parse(mgmtEnv)
	if e != nil {
		return fmt.Errorf("MGMT environ parse error %v", e)
	}

	var addr string
	switch u.Scheme {
	case "unix":
		addr = u.Path
		os.Remove(addr)
	case "tcp", "tcp4", "tcp6":
		addr = u.Host
	default:
		return fmt.Errorf("unsupported MGMT scheme %s", u.Scheme)
	}

	listener, e = net.Listen(u.Scheme, addr)
	if e != nil {
		return fmt.Errorf("cannot listen on %s %s", u.Scheme, addr)
	}

	isClosing = false
	go serve()
	return nil
}

func serve() {
	for !isClosing {
		conn, e := listener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				continue
			} else {
				break
			}
		}
		go Server.ServeCodec(jsonrpc2.NewServerCodec(conn, Server))
	}
	listener = nil
	isClosing = false
}

func Stop() {
	isClosing = true
}
