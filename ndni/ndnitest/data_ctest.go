package ndnitest

/*
#include "../../csrc/ndni/data.h"
#include "../../csrc/ndni/packet.h"

typedef DataEnc_MetaInfoBuffer(23) MetaInfoBuffer23;

bool
c_DataEnc_PrepareMetaInfo23(MetaInfoBuffer23* metaBuf, ContentType ct, uint32_t freshness, LName finalBlock)
{
	return DataEnc_PrepareMetaInfo(metaBuf, ct, freshness, finalBlock);
}
*/
import "C"
import (
	"crypto/rand"
	"math"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
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
		0625
		07060801`, `42080130 // name
		F000 // unknown-ignored
		140E F000 180103 19020104 1A03080131 // metainfo with unknown-ignored
		1502C0C1 // content
		16031B0100 // siginfo
		1700 // sigvalue
	`)
	defer p.Close()
	require.True(bool(C.Packet_ParseL3(p.npkt)))
	require.EqualValues(ndni.PktData, C.Packet_GetType(p.npkt))
	data = C.Packet_GetDataHdr(p.npkt)
	assert.EqualValues(2, data.name.nComps)
	assert.Equal(bytesFromHex("080142080130"), C.GoBytes(unsafe.Pointer(data.name.value), C.int(data.name.length))) // linearized
	assert.EqualValues(260, data.freshness)

	// invalid: unknown-critical
	p = makePacket(`
		060E
		0703080141 // name
		F100 // unknown-critical
		16031B0100 // siginfo
		1700 // sigvalue
	`)
	defer p.Close()
	assert.False(bool(C.Packet_ParseL3(p.npkt)))

	// invalid: MetaInfo with unknown-critical
	p = makePacket(`
		0613
		0703080141 // name
		1405 F100 180103 // metainfo with unknown-critical
		16031B0100 // siginfo
		1700 // sigvalue
	`)
	defer p.Close()
	assert.False(bool(C.Packet_ParseL3(p.npkt)))
}

func ctestDataEncMinimal(t *testing.T) {
	assert, require := makeAR(t)

	var meta C.MetaInfoBuffer23
	ok := C.c_DataEnc_PrepareMetaInfo23(&meta, an.ContentBlob, 0, C.LName{})
	require.True(bool(ok))

	nameP := ndni.NewPName(ndn.ParseName("/DataEnc/minimal"))
	defer nameP.Free()

	m := makePacket(mbuftestenv.Headroom(256))
	defer m.Close()
	npkt := C.DataEnc_EncodePayload(*(*C.LName)(nameP.Ptr()), C.LName{}, unsafe.Pointer(&meta), m.mbuf)
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

	var meta C.MetaInfoBuffer23
	finalBlock := ndn.NameComponentFrom(an.TtSegmentNameComponent, tlv.NNI(math.MaxUint32+1))
	finalBlockP := ndni.NewPName(ndn.Name{finalBlock})
	defer finalBlockP.Free()
	ok := C.c_DataEnc_PrepareMetaInfo23(&meta, an.ContentKey, 3600_000, *(*C.LName)(finalBlockP.Ptr()))
	require.True(bool(ok))

	nameP := ndni.NewPName(ndn.ParseName("/DataEnc/full"))
	defer nameP.Free()
	content := make([]byte, 512)
	rand.Read(content)

	m := makePacket(mbuftestenv.Headroom(256), content)
	defer m.Close()
	npkt := C.DataEnc_EncodePayload(*(*C.LName)(nameP.Ptr()), *(*C.LName)(finalBlockP.Ptr()), unsafe.Pointer(&meta), m.mbuf)
	assert.Equal(m.npkt, npkt)

	data := ndni.PacketFromPtr(m.Ptr()).ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/DataEnc/full/33=%00%00%00%01%00%00%00%00", data)
	assert.EqualValues(an.ContentKey, data.ContentType)
	assert.Equal(time.Hour, data.Freshness)
	assert.Equal(finalBlock, data.FinalBlock)
	assert.Equal(content, data.Content)
}
