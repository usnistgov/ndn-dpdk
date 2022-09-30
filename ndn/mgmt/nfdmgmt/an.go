package nfdmgmt

import "github.com/usnistgov/ndn-dpdk/ndn"

// TLV-TYPE assigned numbers.
const (
	TtControlParameters = 0x68
	TtFaceID            = 0x69
	TtOrigin            = 0x6F
	TtCost              = 0x6A
	TtFlags             = 0x6C
	TtExpirationPeriod  = 0x6D

	TtControlResponse = 0x65
	TtStatusCode      = 0x66
	TtStatusText      = 0x67
)

// RouteOrigin assigned numbers.
const (
	RouteOriginApp     = 0
	RouteOriginStatic  = 255
	RouteOriginClient  = 65
	RouteOriginAutoReg = 64
)

// Command prefixes.
var (
	PrefixLocalhost = ndn.ParseName("/localhost/nfd")
	PrefixLocalhop  = ndn.ParseName("/localhop/nfd")
)
