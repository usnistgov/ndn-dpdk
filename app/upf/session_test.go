package upf_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/wmnsk/go-pfcp/message"
)

// PFCP session related messages from different SMF implementations:
//
//	phoenix: Open5GCore c52b6d04
//	free5gc: free5GC v3.4.2
const (
	pfcpPhoenixEst = "2132010d000000000000000000000300003c000500ac19c30d0039000d020000000202000002ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000002ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0001007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod = "2134003f0000000000000001000004000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000002ac19c40e0058000101"
	pfcpPhoenixDel = "2136000c000000000000000100000a00"
	pfcpFree5gcEst = "233201d0000000000000000000000500003c000500ac19c3130039000d020000000000000002ac19c31300010063003800020001001d0004000000ff000200240014000100001500090100000004ac19c20700160005047663746c005d0005020a8d0001005f000100006c00040000000100510004000000010051000400000002006d000400000002006d00040000000100010051003800020002001d0004000000ff00020017001400010100160005047663746c005d0005060a8d0001006c00040000000200510004000000010051000400000002006d000400000002006d0004000000010003001f006c000400000001002c0001020004000e002a00010100160005047663746c00030016006c000400000002002c00010200040005002a000100000600350051000400000001003e000102002500020300004000040000000a001f00110600000000000003e800000000000003e80064000111000600350051000400000002003e000102002500020300004000040000000a001f00110600000000000003e800000000000003e8006400011000070020006d0004000000010019000100001a000a0000030d4000000186a0007c00010000070012006d0004000000020019000100007c0001010071000101"
	pfcpFree5gcMod = "233400980000000000000001000007c00039000d020000000000000002ac19c31300090041003800020002001d0004000000ff00020017001400010100160005047663746c005d0005060a8d0001006c00040000000200510004000000010051000400000002000a0032006c000400000002002c000102000b0021002a00010000160005047663746c0054000a010000000002ac19c2140031000100"
)

func TestSessionParserPhoenix(t *testing.T) {
	assert, require := makeAR(t)

	var sess upf.SessionParser
	assert.NoError(sess.EstablishmentRequest(parsePFCP(pfcpPhoenixEst).(*message.SessionEstablishmentRequest)))
	assert.NoError(sess.ModificationRequest(parsePFCP(pfcpPhoenixMod).(*message.SessionModificationRequest)))

	loc, ok := sess.LocatorFields()
	require.True(ok)
	assert.EqualValues(0x10000002, loc.UlTEID)
	assert.EqualValues(0x00000002, loc.DlTEID)
	assert.EqualValues(1, loc.UlQFI)
	assert.EqualValues(1, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.196.14"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.1"), loc.InnerRemoteIP)
}

func TestSessionParserFree5gc(t *testing.T) {
	assert, require := makeAR(t)

	var sess upf.SessionParser
	assert.NoError(sess.EstablishmentRequest(parsePFCP(pfcpFree5gcEst).(*message.SessionEstablishmentRequest)))
	assert.NoError(sess.ModificationRequest(parsePFCP(pfcpFree5gcMod).(*message.SessionModificationRequest)))

	loc, ok := sess.LocatorFields()
	require.True(ok)
	assert.EqualValues(0x00000004, loc.UlTEID)
	assert.EqualValues(0x00000002, loc.DlTEID)
	assert.EqualValues(1, loc.UlQFI)
	assert.EqualValues(1, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.194.20"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.1"), loc.InnerRemoteIP)
}

type testSessionTableFaceCreator map[netip.Addr]bool

// CreateFace implements upf.FaceCreator.
func (fc testSessionTableFaceCreator) CreateFace(ctx context.Context, loc upf.SessionLocatorFields) (id string, e error) {
	fc[loc.InnerRemoteIP] = true
	return loc.InnerRemoteIP.String(), nil
}

// DestroyFace implements upf.FaceCreator.
func (fc testSessionTableFaceCreator) DestroyFace(ctx context.Context, id string) error {
	ip, e := netip.ParseAddr(id)
	delete(fc, ip)
	return e
}

func TestSessionTable(t *testing.T) {
	assert, _ := makeAR(t)

	fc := testSessionTableFaceCreator{}
	st := upf.NewSessionTable(fc)

	sess, e := st.EstablishmentRequest(context.TODO(), parsePFCP(pfcpPhoenixEst).(*message.SessionEstablishmentRequest))
	assert.NoError(e)
	assert.NotNil(sess)
	assert.Len(fc, 0)

	msgMod := parsePFCP(pfcpPhoenixMod).(*message.SessionModificationRequest)
	msgMod.Header.SEID = sess.UpSEID
	sessMod, e := st.ModificationRequest(context.TODO(), msgMod)
	assert.NoError(e)
	assert.Same(sess, sessMod)
	assert.Len(fc, 1)

	msgDel := parsePFCP(pfcpPhoenixDel).(*message.SessionDeletionRequest)
	msgDel.Header.SEID = sess.UpSEID
	sessDel, e := st.DeletionRequest(context.TODO(), msgDel)
	assert.NoError(e)
	assert.Same(sess, sessDel)
	assert.Len(fc, 0)
}
