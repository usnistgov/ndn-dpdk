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

	mac := ndn.GetEtherMcastAddr()
	if u.User != nil {
		if mac, e = net.ParseMAC(u.User.Username()); e != nil {
			return e
		}
	}
	if username, e := etherMakeUsername(mac); e != nil {
		return e
	} else {
		u.User = url.User(username)
	}

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

func etherMakeUsername(mac net.HardwareAddr) (string, error) {
	if len(mac) != 6 {
		return "", errors.New("invalid MAC-48 address")
	}
	return fmt.Sprintf("%02X-%02X-%02X-%02X-%02X-%02X", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
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
	username, _ := etherMakeUsername(mac)
	return parseImpl(fmt.Sprintf("ether://%s@%s:%d", username, CleanEthdevName(devName), vid), "MakeEtherUri")
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
