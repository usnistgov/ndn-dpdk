package socketface

import (
	"ndn-dpdk/iface"
)

func init() {
	implByNetwork["udp"] = udpImpl{}
	implByNetwork["udp4"] = udpImpl{}
	implByNetwork["udp6"] = udpImpl{}
	implByNetwork["unixgram"] = unixgramImpl{}
	implByNetwork["tcp"] = tcpImpl{}
	implByNetwork["tcp4"] = tcpImpl{}
	implByNetwork["tcp6"] = tcpImpl{}
	implByNetwork["unix"] = unixImpl{}

	iface.RegisterLocatorType(Locator{}, "udp")
	iface.RegisterLocatorType(Locator{}, "unixgram")
	iface.RegisterLocatorType(Locator{}, "tcp")
	iface.RegisterLocatorType(Locator{}, "unix")
}
