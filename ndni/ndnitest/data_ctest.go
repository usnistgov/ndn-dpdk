package ndnitest

/*
#include "../../csrc/ndni/data.h"
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"crypto/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
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

func ctestDataEncMinimal(t *testing.T) {
	assert, require := makeAR(t)

	var meta C.MetaInfoBuffer
	ok := C.DataEnc_PrepareMetaInfo(&meta, an.ContentBlob, 0, C.LName{})
	require.True(bool(ok))

	nameP := ndni.NewPName(ndn.ParseName("/DataEnc/minimal"))
	defer nameP.Free()

	m := makePacket(mbuftestenv.Headroom(256))
	defer m.Close()
	npkt := C.DataEnc_EncodePayload(*(*C.LName)(nameP.Ptr()), &meta, m.mbuf)
	assert.Equal(m.npkt, npkt)

	data := ndni.PacketFromPtr(m.Ptr()).ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/DataEnc/minimal", data)
	assert.EqualValues(an.ContentBlob, data.ContentType)
	assert.Equal(time.Duration(0), data.Freshness)
	assert.False(data.FinalBlock.Valid())
	assert.Len(data.Content, 0)
	assert.EqualValues(an.SigNull, data.SigInfo.Type)
}

func ctestDataEncFull(t *testing.T) {
	assert, require := makeAR(t)

	var meta C.MetaInfoBuffer
	finalBlockP := ndni.NewPName(ndn.ParseName("/final"))
	defer finalBlockP.Free()
	ok := C.DataEnc_PrepareMetaInfo(&meta, an.ContentKey, 3600_000, *(*C.LName)(finalBlockP.Ptr()))
	require.True(bool(ok))

	nameP := ndni.NewPName(ndn.ParseName("/DataEnc/full"))
	defer nameP.Free()
	content := make([]byte, 512)
	rand.Read(content)

	m := makePacket(mbuftestenv.Headroom(256), content)
	defer m.Close()
	npkt := C.DataEnc_EncodePayload(*(*C.LName)(nameP.Ptr()), &meta, m.mbuf)
	assert.Equal(m.npkt, npkt)

	data := ndni.PacketFromPtr(m.Ptr()).ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/DataEnc/full", data)
	assert.EqualValues(an.ContentKey, data.ContentType)
	assert.Equal(time.Hour, data.Freshness)
	assert.Equal(ndn.ParseNameComponent("final"), data.FinalBlock)
	assert.Equal(content, data.Content)
}
