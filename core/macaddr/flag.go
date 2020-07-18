package macaddr

import (
	"net"
)

// Flag is a flag.Value that wraps net.HardwareAddr.
type Flag struct {
	net.HardwareAddr
}

// Get implements flag.Getter.
func (f *Flag) Get() interface{} {
	return f.HardwareAddr
}

// Set implements flag.Value.
func (f *Flag) Set(s string) (e error) {
	f.HardwareAddr, e = net.ParseMAC(s)
	return
}
