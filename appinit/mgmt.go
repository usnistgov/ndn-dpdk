package appinit

import (
	"net"
	"net/rpc"
	"net/url"
	"os"

	"github.com/powerman/rpc-codec/jsonrpc2"

	"ndn-dpdk/iface/facemgmt"
)

var MgmtRpcServer = rpc.NewServer()

func EnableMgmt() {
	enableFaceMgmt()
}

func enableFaceMgmt() {
	var fm facemgmt.FaceMgmt
	MgmtRpcServer.RegisterName("Faces", fm)
}

var mgmtListener net.Listener

func StartMgmt() {
	if mgmtListener != nil {
		panic("StartMgmt: already started")
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
		Exitf(EXIT_MGMT_ERROR, "StartMgmt: MGMT environ parse error %v", e)
	}

	var addr string
	switch u.Scheme {
	case "unix":
		addr = u.Path
		os.Remove(addr)
	case "tcp", "tcp4", "tcp6":
		addr = u.Host
	default:
		Exitf(EXIT_MGMT_ERROR, "StartMgmt: unsupported MGMT scheme %s", u.Scheme)
	}

	mgmtListener, e = net.Listen(u.Scheme, addr)
	if e != nil {
		Exitf(EXIT_MGMT_ERROR, "StartMgmt: cannot listen on %s %s", u.Scheme, addr)
	}

	go serveMgmtListener()
}

func serveMgmtListener() {
	for {
		conn, e := mgmtListener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				continue
			} else {
				break
			}
		}
		go MgmtRpcServer.ServeCodec(jsonrpc2.NewServerCodec(conn, MgmtRpcServer))
	}
}

func StopMgmt() {
	if mgmtListener == nil {
		return
	}
	mgmtListener.Close()
	mgmtListener = nil
}
