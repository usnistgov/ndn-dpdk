package upf_test

import (
	"context"
	"math/rand/v2"
	"net/netip"
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/wmnsk/go-pfcp/message"
)

// PFCP session related messages from different SMF implementations:
//
//	phoenix: Open5GCore c52b6d04
//	free5gc: free5GC v3.4.2
//	oai no-nrf: OAI-CN5G-SMF v2.0.1, NRF disabled
const (
	pfcpPhoenixEst0 = "2132010d000000000000000000000300003c000500ac19c30d0039000d020000000502000003ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000003ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0002007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod0 = "2134003f0000000000000001000005000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000003ac19c40e0058000101"
	pfcpPhoenixEst1 = "2132010d000000000000000000000400003c000500ac19c30d0039000d020000000702000004ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000004ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0003007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod1 = "2134003f0000000000000002000006000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000004ac19c40e0058000101"
	pfcpPhoenixEst2 = "2132010d000000000000000000000b00003c000500ac19c30d0039000d020000000b02000007ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000007ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0001007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod2 = "2134003f000000000000000300000c000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000007ac19c40e0058000101"
	pfcpPhoenixEst3 = "2132010d000000000000000000000d00003c000500ac19c30d0039000d020000000d02000009ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000009ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0004007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod3 = "2134003f000000000000000400000e000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000009ac19c40e0058000101"
	pfcpFree5gcEst  = "233201d0000000000000000000000500003c000500ac19c3130039000d020000000000000002ac19c31300010063003800020001001d0004000000ff000200240014000100001500090100000004ac19c20700160005047663746c005d0005020a8d0001005f000100006c00040000000100510004000000010051000400000002006d000400000002006d00040000000100010051003800020002001d0004000000ff00020017001400010100160005047663746c005d0005060a8d0001006c00040000000200510004000000010051000400000002006d000400000002006d0004000000010003001f006c000400000001002c0001020004000e002a00010100160005047663746c00030016006c000400000002002c00010200040005002a000100000600350051000400000001003e000102002500020300004000040000000a001f00110600000000000003e800000000000003e80064000111000600350051000400000002003e000102002500020300004000040000000a001f00110600000000000003e800000000000003e8006400011000070020006d0004000000010019000100001a000a0000030d4000000186a0007c00010000070012006d0004000000020019000100007c0001010071000101"
	pfcpFree5gcMod  = "233400980000000000000001000007c00039000d020000000000000002ac19c31300090041003800020002001d0004000000ff00020017001400010100160005047663746c005d0005060a8d0001006c00040000000200510004000000010051000400000002000a0032006c000400000002002c000102000b0021002a00010000160005047663746c0054000a010000000002ac19c2140031000100"
	pfcpOaiNoNrfEst = "213200a70000000000000002bfd1d300003c000500ac19c40a0039000d020000000000000002ac19c40a00010052003800020001001d000400000000000200330014000100001500090100000002ac19c3070016000f06616363657373036f6169036f7267005d0005020a8d0002007c000109005f000100006c00040000000100030027006c000400000001002c00010200040016002a0001010016000d04636f7265036f6169036f7267"
	pfcpOaiNoNrfMod = "213400840000000000000001bfd1d50000010039003800020002001d0004000000000002001f00140001010016000d04636f7265036f6169036f7267005d0005060a8d0002006c00040000000200030037006c000400000002002c00010200040026002a0001000016000f06616363657373036f6169036f72670054000a010000000002ac19c30e"
)

func parseSession(t *testing.T, est, mod string) (sess upf.SessionParser) {
	_, require := makeAR(t)
	_, e := sess.EstablishmentRequest(parsePFCP(est).(*message.SessionEstablishmentRequest), nil)
	require.NoError(e)
	_, e = sess.ModificationRequest(parsePFCP(mod).(*message.SessionModificationRequest), nil)
	require.NoError(e)
	return
}

func TestSessionParserPhoenix(t *testing.T) {
	assert, require := makeAR(t)

	sess := parseSession(t, pfcpPhoenixEst0, pfcpPhoenixMod0)

	loc, ok := sess.LocatorFields()
	require.True(ok)
	assert.EqualValues(0x10000003, loc.UlTEID)
	assert.EqualValues(0x00000003, loc.DlTEID)
	assert.EqualValues(1, loc.UlQFI)
	assert.EqualValues(1, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.196.14"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.2"), loc.InnerRemoteIP)
}

func TestSessionParserFree5gc(t *testing.T) {
	assert, require := makeAR(t)

	sess := parseSession(t, pfcpFree5gcEst, pfcpFree5gcMod)

	loc, ok := sess.LocatorFields()
	require.True(ok)
	assert.EqualValues(0x00000004, loc.UlTEID)
	assert.EqualValues(0x00000002, loc.DlTEID)
	assert.EqualValues(1, loc.UlQFI)
	assert.EqualValues(1, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.194.20"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.1"), loc.InnerRemoteIP)
}

func TestSessionParserOaiNoNrf(t *testing.T) {
	assert, require := makeAR(t)

	sess := parseSession(t, pfcpOaiNoNrfEst, pfcpOaiNoNrfMod)

	loc, ok := sess.LocatorFields()
	require.True(ok)
	assert.EqualValues(0x00000002, loc.UlTEID)
	assert.EqualValues(0x00000002, loc.DlTEID)
	assert.EqualValues(9, loc.UlQFI)
	assert.EqualValues(9, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.195.14"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.2"), loc.InnerRemoteIP)
}

type testSessionTableFaceCreator struct {
	t     testing.TB
	ctx   context.Context
	Table map[string]upf.SessionLocatorFields
}

func (fc *testSessionTableFaceCreator) CreateFace(ctx context.Context, loc upf.SessionLocatorFields) (id string, e error) {
	assert, _ := makeAR(fc.t)
	assert.Same(fc.ctx, ctx)

	id = loc.InnerRemoteIP.String()
	fc.Table[id] = loc
	return id, nil
}

func (fc *testSessionTableFaceCreator) DestroyFace(ctx context.Context, id string) error {
	assert, _ := makeAR(fc.t)
	assert.Same(fc.ctx, ctx)

	delete(fc.Table, id)
	return nil
}

func TestSessionTable(t *testing.T) {
	assert, require := makeAR(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := &testSessionTableFaceCreator{
		t:     t,
		ctx:   ctx,
		Table: map[string]upf.SessionLocatorFields{},
	}
	st := upf.NewSessionTable(fc.CreateFace, fc.DestroyFace)

	establishment := func(wireHex string) *upf.Session {
		sess, _, e := st.EstablishmentRequest(ctx, parsePFCP(wireHex).(*message.SessionEstablishmentRequest), nil)
		require.NoError(e)
		require.NotNil(sess)
		return sess
	}
	modification := func(sess *upf.Session, wireHex string, expectNotExist bool) {
		msg := parsePFCP(wireHex).(*message.SessionModificationRequest)
		msg.Header.SEID = sess.UpSEID
		sessFound, _, e := st.ModificationRequest(ctx, msg, nil)
		if expectNotExist {
			assert.True(os.IsNotExist(e))
			assert.Nil(sessFound)
		} else {
			require.NoError(e)
			assert.Same(sess, sessFound)
		}
	}
	deletion := func(sess *upf.Session, expectNotExist bool) {
		msg := message.NewSessionDeletionRequest(0, 0, sess.UpSEID, rand.Uint32(), 0)
		sessFound, e := st.DeletionRequest(ctx, msg)
		if expectNotExist {
			assert.True(os.IsNotExist(e))
			assert.Nil(sessFound)
		} else {
			require.NoError(e)
			assert.Same(sess, sessFound)
		}
	}

	assert.Len(fc.Table, 0)
	sess0 := establishment(pfcpPhoenixEst0)
	assert.Len(fc.Table, 0)
	sess1 := establishment(pfcpPhoenixEst1)
	assert.Len(fc.Table, 0)
	modification(sess0, pfcpPhoenixMod0, false)
	assert.Len(fc.Table, 1)
	modification(sess1, pfcpPhoenixMod1, false)
	assert.Len(fc.Table, 2)
	sess2 := establishment(pfcpPhoenixEst2)
	assert.Len(fc.Table, 2)
	modification(sess2, pfcpPhoenixMod2, false)
	assert.Len(fc.Table, 3)
	sess3 := establishment(pfcpPhoenixEst3)
	assert.Len(fc.Table, 3)
	modification(sess3, pfcpPhoenixMod3, false)
	assert.Len(fc.Table, 4)

	assert.Equal(upf.SessionLocatorFields{ // sess0
		UlTEID:        0x10000003,
		DlTEID:        0x00000003,
		UlQFI:         1,
		DlQFI:         1,
		RemoteIP:      netip.MustParseAddr("172.25.196.14"),
		InnerRemoteIP: netip.MustParseAddr("10.141.0.2"),
	}, fc.Table["10.141.0.2"])

	assert.Equal(upf.SessionLocatorFields{ // sess1
		UlTEID:        0x10000004,
		DlTEID:        0x00000004,
		UlQFI:         1,
		DlQFI:         1,
		RemoteIP:      netip.MustParseAddr("172.25.196.14"),
		InnerRemoteIP: netip.MustParseAddr("10.141.0.3"),
	}, fc.Table["10.141.0.3"])

	assert.Equal(upf.SessionLocatorFields{ // sess2
		UlTEID:        0x10000007,
		DlTEID:        0x00000007,
		UlQFI:         1,
		DlQFI:         1,
		RemoteIP:      netip.MustParseAddr("172.25.196.14"),
		InnerRemoteIP: netip.MustParseAddr("10.141.0.1"),
	}, fc.Table["10.141.0.1"])

	assert.Equal(upf.SessionLocatorFields{ // sess3
		UlTEID:        0x10000009,
		DlTEID:        0x00000009,
		UlQFI:         1,
		DlQFI:         1,
		RemoteIP:      netip.MustParseAddr("172.25.196.14"),
		InnerRemoteIP: netip.MustParseAddr("10.141.0.4"),
	}, fc.Table["10.141.0.4"])

	deletion(sess3, false)
	assert.Len(fc.Table, 3)
	deletion(sess0, false)
	assert.Len(fc.Table, 2)
	deletion(sess1, false)
	assert.Len(fc.Table, 1)
	deletion(sess2, false)
	assert.Len(fc.Table, 0)

	modification(sess3, pfcpPhoenixMod3, true)
	deletion(sess3, true)
}
