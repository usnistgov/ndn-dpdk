//go:build js && wasm

package wasmtransport

import (
	"io"
	"syscall/js"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

var (
	gArrayBuffer = js.Global().Get("ArrayBuffer")
	gUint8Array  = js.Global().Get("Uint8Array")
	gWebSocket   = js.Global().Get("WebSocket")
)

// NewWebSocket creates a WebSocket transport.
func NewWebSocket(uri string) (l3.Transport, error) {
	sock := gWebSocket.New(uri, []any{})
	sock.Set("binaryType", "arraybuffer")
	tr := &wsTransport{
		sock: sock,
		rxq:  make(chan js.Value, 64),
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU:          8800,
		InitialState: l3.TransportDown,
	})

	tr.handleOpenFunc = js.FuncOf(tr.handleOpen)
	tr.handleCloseErrorFunc = js.FuncOf(tr.handleCloseError)
	tr.handleMessageFunc = js.FuncOf(tr.handleMessage)
	sock.Call("addEventListener", "open", tr.handleOpenFunc)
	sock.Call("addEventListener", "close", tr.handleCloseErrorFunc)
	sock.Call("addEventListener", "error", tr.handleCloseErrorFunc)
	sock.Call("addEventListener", "message", tr.handleMessageFunc)
	return tr, nil
}

type wsTransport struct {
	*l3.TransportBase
	p                    *l3.TransportBasePriv
	sock                 js.Value
	rxq                  chan js.Value
	handleOpenFunc       js.Func
	handleCloseErrorFunc js.Func
	handleMessageFunc    js.Func
}

func (tr *wsTransport) handleOpen(this js.Value, args []js.Value) any {
	tr.p.SetState(l3.TransportUp)
	return nil
}

func (tr *wsTransport) handleCloseError(this js.Value, args []js.Value) any {
	tr.p.SetState(l3.TransportClosed)
	close(tr.rxq)
	return nil
}

func (tr *wsTransport) handleMessage(this js.Value, args []js.Value) any {
	if data := args[0].Get("data"); data.InstanceOf(gArrayBuffer) {
		select {
		case tr.rxq <- data:
		default:
		}
	}
	return nil
}

func (tr *wsTransport) Read(buf []byte) (n int, e error) {
	data, ok := <-tr.rxq
	if !ok {
		return 0, io.EOF
	}

	u8 := gUint8Array.New(data)
	return js.CopyBytesToGo(buf, u8), nil
}

func (tr *wsTransport) Write(buf []byte) (n int, e error) {
	if tr.State() != l3.TransportUp {
		return 0, nil
	}

	u8 := gUint8Array.New(len(buf))
	js.CopyBytesToJS(u8, buf)
	tr.sock.Call("send", u8)
	return len(buf), nil
}

func (tr *wsTransport) Close() error {
	tr.p.SetState(l3.TransportDown)

	wait := make(chan struct{})
	defer tr.OnStateChange(func(st l3.TransportState) {
		if st == l3.TransportClosed {
			close(wait)
		}
	})()
	tr.sock.Call("close")
	<-wait

	tr.handleOpenFunc.Release()
	tr.handleCloseErrorFunc.Release()
	tr.handleMessageFunc.Release()
	return nil
}
