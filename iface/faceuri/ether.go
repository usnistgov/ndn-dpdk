package faceuri

import (
	"errors"
	"fmt"
	"net"
)

type etherImpl struct{}

func (impl etherImpl) Verify(u *FaceUri) error {
	e := rejectUPQF(u)
	if e != nil {
		return e
	}

	if u.Port() != "" {
		return errors.New("ether URI cannot have port number")
	}

	mac, _ := net.ParseMAC(u.Hostname())
	if len(mac) != 6 {
		return fmt.Errorf("ether URI must contain MAC-48 address")
	}
	u.Host = fmt.Sprintf("[%s]", mac)

	return nil
}

func init() {
	implByScheme["ether"] = etherImpl{}
}
