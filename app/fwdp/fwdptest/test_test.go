package fwdptest

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	testenv.Exit(m.Run())
}

var makeAR = testenv.MakeAR

var lastTestToken uint32

type testToken []byte

func (token testToken) LpL3() ndn.LpL3 {
	return ndn.LpL3{PitToken: []byte(token)}
}

func makeToken() (token testToken) {
	token = make(testToken, 24)
	rand.Read([]byte(token[8:]))
	lastTestToken++
	binary.BigEndian.PutUint32([]byte(token), lastTestToken)
	return
}
