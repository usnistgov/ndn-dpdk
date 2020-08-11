package macaddr

import (
	"net"
)

// Flag is a wrapper of net.HardwareAddr compatible with flag and json packages.
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

// MarshalText implements encoding.TextMarshaler.
func (f Flag) MarshalText() (text []byte, e error) {
	return []byte(f.HardwareAddr.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (f *Flag) UnmarshalText(text []byte) (e error) {
	return f.Set(string(text))
}
