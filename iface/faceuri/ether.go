package faceuri

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"ndn-dpdk/ndn"
)

type etherImpl struct{}

func (etherImpl) Verify(u *FaceUri) (e error) {
	if e = u.verifyNo(no.password, no.path, no.query, no.fragment); e != nil {
		return e
	}

	mac := MacAddress(ndn.GetEtherMcastAddr())
	if u.User != nil {
		if mac, e = ParseMacAddress(u.User.Username()); e != nil {
			return e
		}
	}
	u.User = url.User(mac.String())

	devName := u.Hostname()
	if devName == "" {
		return errors.New("empty device name")
	}
	devName = CleanEthdevName(devName)

	vid := 0
	if u.Port() != "" {
		if vid, e = strconv.Atoi(u.Port()); e != nil || !etherCheckVlan(vid) {
			return errors.New("invalid VLAN identifier")
		}
	}
	u.Host = net.JoinHostPort(devName, strconv.Itoa(vid))

	return nil
}

func etherCheckVlan(vid int) bool {
	return vid >= 0x000 && vid < 0xfff
}

func init() {
	implByScheme["ether"] = etherImpl{}
}

// Clean Ethernet device name so that it's usable as hostname.
func CleanEthdevName(input string) string {
	return strings.Map(func(c rune) rune {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			return c
		}
		return '-'
	}, input)
}

// Construct ether FaceUri from device name, MAC address (nil for default multicast),
// and VLAN identifier (0 for default VLAN).
func MakeEtherUri(devName string, mac net.HardwareAddr, vid int) (*FaceUri, error) {
	if len(mac) == 0 {
		mac = ndn.GetEtherMcastAddr()
	}
	mac2 := MacAddress(mac)
	if !mac2.Valid() {
		return nil, errInvalidMacAddress
	}
	return parseImpl(fmt.Sprintf("ether://%s@%s:%d", mac, CleanEthdevName(devName), vid), "MakeEtherUri")
}

func MustMakeEtherUri(devName string, mac net.HardwareAddr, vid int) *FaceUri {
	u, e := MakeEtherUri(devName, mac, vid)
	if e != nil {
		panic(e)
	}
	return u
}

// Extract device name, MAC address, and VLAN identifier from ether FaceUri.
// Undefined behavior may occur if u is not a valid FaceUri of ether scheme.
func (u *FaceUri) ExtractEther() (devName string, mac net.HardwareAddr, vid int) {
	devName = u.Hostname()
	mac, _ = net.ParseMAC(u.User.Username())
	vid, _ = strconv.Atoi(u.Port())
	return
}
