package faceuri

import (
	"errors"
	"net"
	"strconv"

	"ndn-dpdk/ndn"
)

type udpTcpImpl struct {
	scheme      string
	defaultPort uint16
}

func (impl udpTcpImpl) Verify(u *FaceUri) (e error) {
	if e = u.verifyNo(no.user, no.path, no.query, no.fragment); e != nil {
		return e
	}

	ip := net.ParseIP(u.Hostname()).To4()
	if ip == nil || ip[0] < 1 || ip[0] > 223 {
		return errors.New("not an IPv4 unicast address")
	}

	port := int(impl.defaultPort)
	if u.Port() != "" {
		if port, e = strconv.Atoi(u.Port()); e != nil || !impl.checkPort(port) {
			return errors.New("invalid port number")
		}
	}

	u.Host = net.JoinHostPort(ip.String(), strconv.Itoa(port))
	u.Scheme = impl.scheme
	return nil
}

func (udpTcpImpl) checkPort(port int) bool {
	return port > 0 && port <= 65535
}

func init() {
	implByScheme["udp4"] = udpTcpImpl{"udp4", ndn.NDN_UDP_PORT}
	implByScheme["udp"] = implByScheme["udp4"]
	implByScheme["tcp4"] = udpTcpImpl{"tcp4", ndn.NDN_TCP_PORT}
	implByScheme["tcp"] = implByScheme["tcp4"]
}
