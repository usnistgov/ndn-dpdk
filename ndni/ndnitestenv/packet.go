package ndnitestenv

import (
	"reflect"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func makePacket(b []byte) *ndni.Packet {
	m := mbuftestenv.MakePacket(b)
	m.SetTimestamp(eal.TscNow())
	return ndni.PacketFromPtr(m.GetPtr())
}

func parseL2L3(pkt *ndni.Packet) {
	e := pkt.ParseL2()
	if e != nil {
		panic(e)
	}

	e = pkt.ParseL3(Name.Pool())
	if e != nil {
		panic(e)
	}
}

// ActiveFHDelegation selects an active forwarding hint delegation.
type ActiveFHDelegation int

// SetActiveFH creates ActiveFHDelegation.
func SetActiveFH(index int) ActiveFHDelegation {
	return ActiveFHDelegation(index)
}

// MakeInterest creates an Interest packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: arguments to ndn.MakeInterest (valid if input is name URI), or ActiveFHDelegation.
// Panics if packet constructed from bytes is not Interest.
func MakeInterest(input interface{}, args ...interface{}) (interest *ndni.Interest) {
	activeFh := -1
	var nArgs []interface{}
	for _, arg := range args {
		switch a := arg.(type) {
		case ActiveFHDelegation:
			activeFh = int(a)
		default:
			nArgs = append(nArgs, arg)
		}
	}

	var pkt *ndni.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			nArgs = append(nArgs, inp)
			nInterest := ndn.MakeInterest(nArgs...)
			wire, e := tlv.Encode(nInterest)
			if e != nil {
				panic(e)
			}
			pkt = makePacket(wire)
		} else {
			pkt = makePacket(testenv.BytesFromHex(inp))
		}
	default:
		panic("bad argument type " + reflect.TypeOf(input).String())
	}

	parseL2L3(pkt)
	interest = pkt.AsInterest()
	if activeFh >= 0 {
		if e := interest.SelectActiveFh(activeFh); e != nil {
			panic(e)
		}
	}
	return interest
}

// MakeData creates a Data packet.
// input: packet bytes as []byte or HEX, or name URI.
// args: arguments to ndn.MakeData (valid if input is name URI).
// Panics if packet constructed from bytes is not Data.
func MakeData(input interface{}, args ...interface{}) *ndni.Data {
	var pkt *ndni.Packet
	switch inp := input.(type) {
	case []byte:
		pkt = makePacket(inp)
	case string:
		if inp[0] == '/' {
			nArgs := append([]interface{}{inp}, args...)
			nData := ndn.MakeData(nArgs...)
			wire, e := tlv.Encode(nData)
			if e != nil {
				panic(e)
			}
			pkt = makePacket(wire)
		} else {
			pkt = makePacket(testenv.BytesFromHex(inp))
		}
	default:
		panic("bad argument type " + reflect.TypeOf(input).String())
	}

	parseL2L3(pkt)
	return pkt.AsData()
}

// SetPort updates mbuf.port field.
func SetPort(pkt ndni.IL3Packet, port uint16) {
	pkt.GetPacket().AsMbuf().SetPort(port)
}

// GetPitToken returns the PIT token.
func GetPitToken(pkt ndni.IL3Packet) uint64 {
	return pkt.GetPacket().GetLpL3().PitToken
}

// SetPitToken updates the PIT token.
func SetPitToken(pkt ndni.IL3Packet, token uint64) {
	pkt.GetPacket().GetLpL3().PitToken = token
}

// CopyPitToken copies PIT token from src to pkt.
func CopyPitToken(pkt ndni.IL3Packet, src ndni.IL3Packet) {
	SetPitToken(pkt, GetPitToken(src))
}

// ClosePacket releases the mbuf.
func ClosePacket(pkt ndni.IL3Packet) {
	pkt.GetPacket().AsMbuf().Close()
}
