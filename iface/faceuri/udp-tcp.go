package faceuri

import (
	"fmt"
	"net"

	"ndn-dpdk/ndn"
)

type udpTcpImpl struct {
	scheme      string
	defaultPort uint16
}

func (impl udpTcpImpl) Verify(u *FaceUri) error {
	e := rejectUPQF(u)
	if e != nil {
		return e
	}

	ip := net.ParseIP(u.Hostname()).To4()
	if ip == nil || ip[0] < 1 || ip[0] > 223 {
		return fmt.Errorf("%s URI must contain IPv4 unicast address", u.Scheme)
	}

	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Host, fmt.Sprintf("%d", impl.defaultPort))
	} else {
		var portNo uint16
		_, e := fmt.Sscan(u.Port(), &portNo)
		if e != nil {
			return fmt.Errorf("%s URI needs a valid port number but %s has error %v",
				u.Scheme, u.Port(), e)
		}
		if portNo == 0 {
			return fmt.Errorf("%s URI cannot have port number 0", u.Scheme)
		}
	}

	u.Scheme = impl.scheme
	return nil
}

func init() {
	implByScheme["udp4"] = udpTcpImpl{"udp4", ndn.NDN_UDP_PORT}
	implByScheme["udp"] = implByScheme["udp4"]
	implByScheme["tcp4"] = udpTcpImpl{"tcp4", ndn.NDN_TCP_PORT}
	implByScheme["tcp"] = implByScheme["tcp4"]
}
