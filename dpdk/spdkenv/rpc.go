package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/rpc.h>

int c_spdk_rpc_accept(void* arg)
{
	spdk_rpc_accept();
	return -1;
}
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"github.com/phayes/freeport"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

var (
	rpcClient *jsonrpc2.Client
	rpcPoller *Poller
)

// Enable SPDK RPC server and internal RPC client.
func initRPC() error {
	port, e := freeport.GetFreePort()
	if e != nil {
		return fmt.Errorf("TCP listen port unavailable: %w", e)
	}

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	listenAddrC := C.CString(listenAddr)
	defer C.free(unsafe.Pointer(listenAddrC))

	res := C.spdk_rpc_listen(listenAddrC)
	if res != 0 {
		return fmt.Errorf("spdk_rpc_listen error on %s", listenAddr)
	}
	rpcPoller = NewPoller(mainThread, cptr.Func0.C(C.c_spdk_rpc_accept, nil), 10*time.Millisecond)

	rpcClient, e = jsonrpc2.Dial("tcp", listenAddr)
	if e != nil {
		return fmt.Errorf("jsonrpc2.Dial error: %w", e)
	}

	return nil
}

// RPC calls a method on SPDK RPC server.
func RPC(method string, args interface{}, reply interface{}) error {
	return rpcClient.Call(method, args, reply)
}
