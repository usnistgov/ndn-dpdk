package upf_test

import (
	"encoding/hex"
	"net/netip"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/wmnsk/go-pfcp/message"
)

const (
	pfcpPhoenixEst = "2132010d000000000000000000000300003c000500ac19c30d0039000d020000000202000002ac19c30d0001003e003800020456001d0004000000ff000200170014000100001500090110000002ac19c407007c000101005f000100006c00040000006e006d0004000004560001003e0038000203f2001d0004000000ff0002001c001400010100160005047663746c005d0005060a8d0001007c000101006c00040000000a006d0004000003f200030025006c00040000006e002c0001020004000f002a000101001600066e365f74756e005800010100070012006d000400000456007c000101001900010000070012006d0004000003f2007c00010100190001000055000500580001010071000101"
	pfcpPhoenixMod = "2134003f0000000000000001000004000003002f006c00040000000a002c00010200040019002a000100001600026e330054000a010000000002ac19c40e0058000101"
)

func sessionEstabMod(t *testing.T, sess *upf.Session, estHex, modHex string) {
	assert, require := makeAR(t)

	{
		wire, e := hex.DecodeString(estHex)
		require.NoError(e)
		msg, e := message.ParseSessionEstablishmentRequest(wire)
		require.NoError(e)
		e = sess.EstablishmentRequest(msg)
		assert.NoError(e)
	}

	{
		wire, e := hex.DecodeString(modHex)
		require.NoError(e)
		msg, e := message.ParseSessionModificationRequest(wire)
		require.NoError(e)
		e = sess.ModificationRequest(msg)
		assert.NoError(e)
	}
}

func TestSessionPhoenix(t *testing.T) {
	assert, _ := makeAR(t)
	var sess upf.Session
	sessionEstabMod(t, &sess, pfcpPhoenixEst, pfcpPhoenixMod)
	loc, ok := sess.LocatorFields()
	assert.True(ok)
	assert.EqualValues(0x10000002, loc.UlTEID)
	assert.EqualValues(0x00000002, loc.DlTEID)
	assert.EqualValues(1, loc.UlQFI)
	assert.EqualValues(1, loc.DlQFI)
	assert.EqualValues(netip.MustParseAddr("172.25.196.14"), loc.RemoteIP)
	assert.EqualValues(netip.MustParseAddr("10.141.0.1"), loc.InnerRemoteIP)
}
