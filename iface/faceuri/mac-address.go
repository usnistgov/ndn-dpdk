package faceuri

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
)

type MacAddress net.HardwareAddr

var errInvalidMacAddress = errors.New("invalid MAC-48 address")

func ParseMacAddress(input string) (a MacAddress, e error) {
	aa, e := net.ParseMAC(input)
	if e != nil {
		return nil, e
	}
	a = MacAddress(aa)
	if !a.Valid() {
		return nil, errInvalidMacAddress
	}
	return a, nil
}

func (a MacAddress) Valid() bool {
	return len(a) == 6
}

func (a MacAddress) String() string {
	if !a.Valid() {
		return "00-00-00-00-00-00"
	}
	return fmt.Sprintf("%02X-%02X-%02X-%02X-%02X-%02X", a[0], a[1], a[2], a[3], a[4], a[5])
}

func (a MacAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *MacAddress) UnmarshalJSON(data []byte) error {
	return a.UnmarshalYAML(func(v interface{}) error {
		return json.Unmarshal(data, v)
	})
}

func (a MacAddress) MarshalYAML() (interface{}, error) {
	return a.String(), nil
}

func (a *MacAddress) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if e := unmarshal(&raw); e != nil {
		return e
	}
	if raw == "" {
		return nil
	}
	if a2, e := ParseMacAddress(raw); e != nil {
		return e
	} else {
		*a = a2
	}
	return nil
}
