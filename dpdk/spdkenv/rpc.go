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
	"os"
	"time"
	"unsafe"

	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

var (
	rpcClient *jsonrpc2.Client
	rpcPoller *Poller
)

// Enable SPDK RPC server and internal RPC client.
func initRPC() error {
	dir, e := os.MkdirTemp("", "spdk-*")
	if e != nil {
		return fmt.Errorf("Unix socket path unavailable: %w", e)
	}
	defer os.RemoveAll(dir)

	sockName := dir + "/spdk.sock"
	sockNameC := C.CString(sockName)
	defer C.free(unsafe.Pointer(sockNameC))

	res := C.spdk_rpc_listen(sockNameC)
	if res != 0 {
		return fmt.Errorf("spdk_rpc_listen error on %s", sockName)
	}
	rpcPoller = NewPoller(mainThread, cptr.Func0.C(C.c_spdk_rpc_accept, nil), 10*time.Millisecond)

	rpcClient, e = jsonrpc2.Dial("unix", sockName)
	if e != nil {
		return fmt.Errorf("jsonrpc2.Dial error: %w", e)
	}

	return nil
}

// RPC calls a method on SPDK RPC server.
func RPC(method string, args interface{}, reply interface{}) error {
	return rpcClient.Call(method, args, reply)
}
