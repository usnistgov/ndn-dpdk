package ndn

/*
#include "protonum.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk"
)

const (
	NDN_ETHERTYPE = uint16(C.NDN_ETHERTYPE)
	NDN_UDP_PORT  = uint16(C.NDN_UDP_PORT)
	NDN_TCP_PORT  = uint16(C.NDN_TCP_PORT)
	NDN_WS_PORT   = uint16(C.NDN_WS_PORT)
)

var NDN_ETHER_MCAST_ADDR dpdk.EtherAddr

func init() {
	NDN_ETHER_MCAST_ADDR, _ = dpdk.ParseEtherAddr("01:00:5E:00:17:AA")
}
