package ndn

/*
#include "protonum.h"
*/
import "C"
import (
	"net"
)

const (
	NDN_ETHERTYPE = uint16(C.NDN_ETHERTYPE)
	NDN_UDP_PORT  = uint16(C.NDN_UDP_PORT)
	NDN_TCP_PORT  = uint16(C.NDN_TCP_PORT)
	NDN_WS_PORT   = uint16(C.NDN_WS_PORT)
)

func GetEtherMcastAddr() net.HardwareAddr {
	return net.HardwareAddr{0x01, 0x00, 0x5E, 0x00, 0x17, 0xAA}
}
