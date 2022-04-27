package macaddr_test

import (
	"encoding/json"
	"flag"
	"net"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var makeAR = testenv.MakeAR

func TestMacAddr(t *testing.T) {
	assert, _ := makeAR(t)

	macZero, _ := net.ParseMAC("00:00:00:00:00:00")
	uA1, _ := net.ParseMAC("02:00:00:00:00:A1")
	uA2, _ := net.ParseMAC("02:00:00:00:00:A2")
	mA1, _ := net.ParseMAC("03:00:00:00:00:A1")
	mac64, _ := net.ParseMAC("02:00:00:00:00:00:00:64")

	assert.True(macaddr.Equal(uA1, uA1))
	assert.False(macaddr.Equal(uA1, uA2))
	assert.False(macaddr.Equal(uA1, mA1))

	assert.False(macaddr.IsUnicast(macZero))
	assert.True(macaddr.IsUnicast(uA1))
	assert.False(macaddr.IsUnicast(mA1))
	assert.False(macaddr.IsUnicast(mac64))

	assert.False(macaddr.IsMulticast(macZero))
	assert.False(macaddr.IsMulticast(uA1))
	assert.True(macaddr.IsMulticast(mA1))
	assert.False(macaddr.IsMulticast(mac64))
}

func TestMakeRandom(t *testing.T) {
	assert, _ := makeAR(t)

	for i := 0; i < 100; i++ {
		a := macaddr.MakeRandomUnicast()
		assert.True(macaddr.IsUnicast(a))
	}
}

func TestFlag(t *testing.T) {
	assert, _ := makeAR(t)

	var f flag.FlagSet
	var m macaddr.Flag
	f.Var(&m, "m", "")
	assert.True(m.Empty())

	assert.Error(f.Parse([]string{"-m", "x"}))
	assert.NoError(f.Parse([]string{"-m", "02:00:00:00:00:A0"}))

	assert.False(m.Empty())
	assert.Equal(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0xA0}, m.Get())
}

func TestJSON(t *testing.T) {
	assert, _ := makeAR(t)

	var m macaddr.Flag
	assert.NoError(json.Unmarshal([]byte("\"02:00:00:00:00:A0\""), &m))

	j, e := json.Marshal(m)
	assert.NoError(e)
	assert.Equal("\"02:00:00:00:00:a0\"", string(j))
}
