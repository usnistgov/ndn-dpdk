package faceuri

import (
	"fmt"
	"net"
)

type etherImpl struct{}

func (impl etherImpl) Verify(u *FaceUri) error {
	if e := u.verifyNo(no.user, no.port, no.path, no.query, no.fragment); e != nil {
		return e
	}

	mac, _ := net.ParseMAC(u.Hostname())
	if len(mac) != 6 {
		return fmt.Errorf("ether FaceUri must contain MAC-48 address")
	}
	u.Host = fmt.Sprintf("[%s]", mac)

	return nil
}

func init() {
	implByScheme["ether"] = etherImpl{}
}
