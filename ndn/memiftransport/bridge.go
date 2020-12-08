package memiftransport

import (
	"errors"
	"fmt"
)

// Bridge bridges two memif interfaces.
// The memifs can operate in either server or client mode.
//
// This is mainly useful for unit testing.
// It is impossible to run both memif peers in the same process, so the test program should execute this bridge in a separate process.
type Bridge struct {
	hdlA    *handle
	hdlB    *handle
	closing chan bool
}

// NewBridge creates a Bridge.
func NewBridge(locA, locB Locator, role Role) (bridge *Bridge, e error) {
	if e = locA.Validate(); e != nil {
		return nil, fmt.Errorf("LocatorA %w", e)
	}
	locA.ApplyDefaults()
	if e = locB.Validate(); e != nil {
		return nil, fmt.Errorf("LocatorB %w", e)
	}
	locB.ApplyDefaults()
	if role == RoleServer && locA.SocketName == locB.SocketName {
		return nil, errors.New("Locators must use different SocketName")
	}

	bridge = &Bridge{
		closing: make(chan bool, 2),
	}
	bridge.hdlA, e = newHandle(locA, role)
	if e != nil {
		return nil, fmt.Errorf("newHandleA %w", e)
	}
	bridge.hdlB, e = newHandle(locB, role)
	if e != nil {
		bridge.hdlA.Close()
		return nil, fmt.Errorf("newHandleB %w", e)
	}

	go bridge.transferLoop(bridge.hdlA, bridge.hdlB)
	go bridge.transferLoop(bridge.hdlB, bridge.hdlA)
	return bridge, nil
}

func (bridge *Bridge) transferLoop(src, dst *handle) {
	macBuffer := make([]byte, 6)
	for {
		select {
		case <-bridge.closing:
			return
		default:
		}

		data, ci, e := src.ReadPacketData()
		if e != nil || ci.CaptureLength < 14 {
			continue
		}

		copy(macBuffer, data[0:6])
		copy(data[0:6], data[6:12])
		copy(data[6:12], macBuffer)
		dst.WritePacketData(data)
	}
}

// Close stops the bridge.
func (bridge *Bridge) Close() error {
	bridge.closing <- true
	bridge.closing <- true
	bridge.hdlA.Close()
	bridge.hdlB.Close()
	return nil
}
