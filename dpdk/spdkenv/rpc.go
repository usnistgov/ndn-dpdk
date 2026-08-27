package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/init.h>
#include <spdk/rpc.h>
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"unsafe"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var rpcClient *jrpc2.Client

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

	res := C.spdk_rpc_initialize(sockNameC, nil)
	if res != 0 {
		return fmt.Errorf("spdk_rpc_initialize error: %w", eal.MakeErrno(res))
	}
	C.spdk_rpc_set_state(C.SPDK_RPC_RUNTIME)

	conn, e := net.Dial("unix", sockName)
	if e != nil {
		return fmt.Errorf("net.Dial error: %w", e)
	}
	rpcClient = jrpc2.NewClient(channel.Line(conn, conn), nil)

	return nil
}

// RPC calls a method on SPDK RPC server.
func RPC(method string, args, reply any) (e error) {
	e = rpcClient.CallResult(context.Background(), method, args, &reply)

	if ce := logger.Check(zap.DebugLevel, "RPC"); ce != nil {
		errField := zap.Skip()
		if e != nil {
			var errV any
			if json.Unmarshal([]byte(e.Error()), &errV) == nil {
				errField = zap.Any("error", errV)
			} else {
				errField = zap.Error(e)
			}
		}
		ce.Write(
			zap.String("method", method),
			zap.Any("args", args),
			zap.Any("reply", reply),
			errField,
		)
	}

	return e
}
