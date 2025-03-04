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
	st := upf.NewSessionTable(netip.MustParseAddr("192.168.3.2"), fc.CreateFace, fc.DestroyFace)

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
