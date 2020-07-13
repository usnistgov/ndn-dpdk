package ndnitest

/*
#include "../../csrc/ndni/data.h"
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

func ctestDataParse(t *testing.T) {
	assert, require := makeAR(t)

	// minimal
	p := makePacket(`
		060C
		0703050141 // name
		16031B0100 // siginfo
		1700 // sigvalue
	`)
	defer p.Close()
	require.True(bool(C.Packet_Parse(p.npkt)))
	require.EqualValues(ndni.PktData, C.Packet_GetType(p.npkt))
	data := C.Packet_GetDataHdr(p.npkt)
	assert.EqualValues(1, data.name.nComps)
	assert.Equal(bytesFromHex("050141"), C.GoBytes(unsafe.Pointer(data.name.value), C.int(data.name.length)))
	assert.EqualValues(0, data.freshness)

	// full
	p = makePacket(`
		0623
		07060801`, `42080130 // name
		140C 180103 19020104 1A03080131 // metainfo
		1502C0C1 // content
		16031B0100 // siginfo
		F000 // unknown-ignored
		1700 // sigvalue
	`)
	require.True(bool(C.Packet_ParseL3(p.npkt)))
	require.EqualValues(ndni.PktData, C.Packet_GetType(p.npkt))
	data = C.Packet_GetDataHdr(p.npkt)
	assert.EqualValues(2, data.name.nComps)
	assert.Equal(bytesFromHex("080142080130"), C.GoBytes(unsafe.Pointer(data.name.value), C.int(data.name.length)))
	assert.EqualValues(260, data.freshness)
}
