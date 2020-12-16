// Package ndnitestenv contains helper functions to construct NDN packets in test code.
package ndnitestenv

/*
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"reflect"
	"strings"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// MakePacketHeadroom is the mbuf headroom used in MakePacket().
var MakePacketHeadroom = mbuftestenv.Headroom(pktmbuf.DefaultHeadroom)

// MakePacket creates a packet.
// input: packet bytes as []byte or HEX.
// modifiers: optional PacketModifiers.
func MakePacket(input interface{}, modifiers ...PacketModifier) *ndni.Packet {
	var b []byte
	switch inp := input.(type) {
	case []byte:
		b = inp
	case string:
		b = testenv.BytesFromHex(inp)
	default:
		panic("bad argument type " + reflect.TypeOf(input).String())
	}

	m := mbuftestenv.MakePacket(b, MakePacketHeadroom)
	m.SetTimestamp(eal.TscNow())

	pkt := ndni.PacketFromPtr(m.Ptr())
	ok := C.Packet_Parse((*C.Packet)(pkt.Ptr()))
	if !bool(ok) {
		panic("C.Packet_Parse error")
	}

	for _, m := range modifiers {
		m.modify(pkt)
	}
	return pkt
}

// MakeInterest creates an Interest packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: arguments to ndn.MakeInterest (valid if input is name URI), or PacketModifiers.
func MakeInterest(input interface{}, args ...interface{}) (pkt *ndni.Packet) {
	modifiers, mArgs := filterPacketModifers(args)
	if s, ok := input.(string); ok && strings.HasPrefix(s, "/") {
		interest := ndn.MakeInterest(append(mArgs, s)...)
		wire, e := tlv.Encode(interest)
		if e != nil {
			panic(e)
		}
		return MakePacket(wire, modifiers...)
	}
	return MakePacket(input, modifiers...)
}

// MakeData creates a Data packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: arguments to ndn.MakeData (valid if input is name URI), or PacketModifiers.
// Panics if packet constructed from bytes is not Data.
func MakeData(input interface{}, args ...interface{}) (pkt *ndni.Packet) {
	modifiers, mArgs := filterPacketModifers(args)
	if s, ok := input.(string); ok && strings.HasPrefix(s, "/") {
		data := ndn.MakeData(append(mArgs, s)...)
		wire, e := tlv.Encode(data)
		if e != nil {
			panic(e)
		}
		return MakePacket(wire, modifiers...)
	}
	return MakePacket(input, modifiers...)
}

// MakeNack turns an Interest to a Nack.
// args: arguments to ndn.MakeNack, or PacketModifiers.
// Note that the Interest must be passed as ndn.Interest instance, not bytes or name.
func MakeNack(args ...interface{}) (pkt *ndni.Packet) {
	modifiers, mArgs := filterPacketModifers(args)
	nack := ndn.MakeNack(mArgs...)
	wire, e := tlv.Encode(nack.ToPacket())
	if e != nil {
		panic(e)
	}
	return MakePacket(wire, modifiers...)
}

// PacketModifier is a function that modifies a created packet.
type PacketModifier interface {
	modify(pkt *ndni.Packet)
}

func filterPacketModifers(args []interface{}) (modifiers []PacketModifier, others []interface{}) {
	for _, arg := range args {
		switch a := arg.(type) {
		case PacketModifier:
			modifiers = append(modifiers, a)
		default:
			others = append(others, arg)
		}
	}
	return
}

// SetActiveFwHint selects an active forwarding hint delegation.
// This applies to Interest only.
func SetActiveFwHint(index int) PacketModifier {
	return modifyActiveFwHint(index)
}

type modifyActiveFwHint int

func (m modifyActiveFwHint) modify(pkt *ndni.Packet) {
	pinterest := C.Packet_GetInterestHdr((*C.Packet)(pkt.Ptr()))
	ok := C.PInterest_SelectFwHint(pinterest, C.int(m))
	if !bool(ok) {
		panic("C.PInterest_SelectFwHint error")
	}
}

// SetPitToken updates PIT token of packet.
func SetPitToken(token uint64) PacketModifier {
	return modifyPitToken(token)
}

type modifyPitToken uint64

func (m modifyPitToken) modify(pkt *ndni.Packet) {
	pkt.SetPitToken(uint64(m))
}

// SetFace updates ingress faceID of packet.
func SetFace(faceID iface.ID) PacketModifier {
	return modifyPort(faceID)
}

type modifyPort uint16

func (m modifyPort) modify(pkt *ndni.Packet) {
	pkt.Mbuf().SetPort(uint16(m))
}
