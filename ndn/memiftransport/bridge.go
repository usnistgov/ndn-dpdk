//go:build linux

package memiftransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"os/exec"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Bridge bridges two memif interfaces.
// The memifs can operate in either server or client mode.
//
// This is mainly useful for unit testing.
// It is impossible to run both memif peers in the same process, so the test program should run this bridge in a separate process.
type Bridge struct {
	hdlA    *handle
	hdlB    *handle
	closing chan struct{}
}

func (bridge *Bridge) transferLoop(src, dst *handle) {
	buf := make([]byte, max(src.loc.Dataroom, dst.loc.Dataroom))
	for {
		select {
		case <-bridge.closing:
			return
		default:
		}

		n, e := src.Read(buf)
		if e == nil && n > 0 {
			dst.Write(buf[:n])
		}
	}
}

// Close stops the bridge.
func (bridge *Bridge) Close() error {
	close(bridge.closing)
	return errors.Join(bridge.hdlA.Close(), bridge.hdlB.Close())
}

// NewBridge creates a Bridge.
func NewBridge(locA, locB Locator) (bridge *Bridge, e error) {
	if e = locA.Validate(); e != nil {
		return nil, fmt.Errorf("locA %w", e)
	}
	locA.ApplyDefaults(RoleServer)
	if e = locB.Validate(); e != nil {
		return nil, fmt.Errorf("locB %w", e)
	}
	locB.ApplyDefaults(RoleServer)

	bridge = &Bridge{
		closing: make(chan struct{}),
	}
	bridge.hdlA, e = newHandle(locA, func(l3.TransportState) {})
	if e != nil {
		return nil, fmt.Errorf("newHandleA %w", e)
	}
	bridge.hdlB, e = newHandle(locB, func(l3.TransportState) {})
	if e != nil {
		bridge.hdlA.Close()
		return nil, fmt.Errorf("newHandleB %w", e)
	}

	go bridge.transferLoop(bridge.hdlA, bridge.hdlB)
	go bridge.transferLoop(bridge.hdlB, bridge.hdlA)
	return bridge, nil
}

const bridgeArg = "986a6a90-4c54-44a0-a585-edee6104d4fa"

// ForkBridgeHelper forks a bridge helper subprocess and invokes f() while it's running.
func ForkBridgeHelper(locA, locB Locator, f func()) error {
	time.Sleep(1 * time.Second)

	locAj, _ := json.Marshal(locA)
	locBj, _ := json.Marshal(locB)

	helper := exec.Command(os.Args[0], bridgeArg, string(locAj), string(locBj))
	helperIn, e := helper.StdinPipe()
	if e != nil {
		return fmt.Errorf("helper.StdinPipe() %w", e)
	}
	helper.Stdout = os.Stdout
	helper.Stderr = os.Stderr
	if e := helper.Start(); e != nil {
		return fmt.Errorf("helper.Start() %w", e)
	}
	defer helper.Process.Kill()
	time.Sleep(1 * time.Second)

	f()

	helperIn.Write([]byte("."))
	if e := helper.Wait(); e != nil {
		return fmt.Errorf("helper.Wait() %w", e)
	}
	return nil
}

// ExecBridgeHelper runs the bridge helper if os.Args requests it.
// In unit test binary, this should be invoked in TestMain() function.
func ExecBridgeHelper() {
	if len(os.Args) == 4 && os.Args[1] == bridgeArg {
		res := runBridgeHelper()
		os.Exit(res)
	}
}

func runBridgeHelper() int {
	stdlog.SetFlags(0)
	stdlog.SetPrefix("memifBridgeHelper ")

	var locA, locB Locator
	if e := json.Unmarshal([]byte(os.Args[2]), &locA); e != nil {
		stdlog.Print("locA", e)
		return 1
	}
	if e := json.Unmarshal([]byte(os.Args[3]), &locB); e != nil {
		stdlog.Print("locB", e)
		return 1
	}

	bridge, e := NewBridge(locA, locB)
	if e != nil {
		stdlog.Print("NewBridge", e)
		return 2
	}
	stdlog.Print("Bridge open")

	io.ReadAtLeast(os.Stdin, make([]byte, 1), 1)
	if e := bridge.Close(); e != nil {
		stdlog.Print("bridge.Close()", e)
		return 3
	}

	stdlog.Print("Bridge closed")
	return 0
}
