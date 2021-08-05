package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/init.h>
#include <spdk/rpc.h>
*/
import "C"
import (
	"fmt"
	"os"
	"path"
	"unsafe"

	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var rpcClient *jsonrpc2.Client

// Enable SPDK RPC server and internal RPC client.
func initRPC() error {
	dir, e := os.MkdirTemp("", "spdk-*")
	if e != nil {
		return fmt.Errorf("unix socket path unavailable: %w", e)
	}
	defer os.RemoveAll(dir)

	sockName := path.Join(dir, "spdk.sock")
	sockNameC := C.CString(sockName)
	defer C.free(unsafe.Pointer(sockNameC))

	res := C.spdk_rpc_initialize(sockNameC)
	if res != 0 {
		return fmt.Errorf("spdk_rpc_initialize error %w", eal.MakeErrno(res))
	}
	C.spdk_rpc_set_state(C.SPDK_RPC_RUNTIME)

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
