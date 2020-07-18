package macaddr_test

import (
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

	assert.True(macaddr.IsValid(macZero))
	assert.True(macaddr.IsValid(uA1))
	assert.True(macaddr.IsValid(mA1))
	assert.False(macaddr.IsValid(mac64))

	assert.False(macaddr.IsUnicast(macZero))
	assert.True(macaddr.IsUnicast(uA1))
	assert.False(macaddr.IsUnicast(mA1))
	assert.False(macaddr.IsUnicast(mac64))

	assert.False(macaddr.IsMulticast(macZero))
	assert.False(macaddr.IsMulticast(uA1))
	assert.True(macaddr.IsMulticast(mA1))
	assert.False(macaddr.IsMulticast(mac64))
}

func TestFlag(t *testing.T) {
	assert, _ := makeAR(t)

	var f flag.FlagSet
	var m macaddr.Flag
	f.Var(&m, "m", "")

	assert.Error(f.Parse([]string{"-m", "x"}))
	assert.NoError(f.Parse([]string{"-m", "02:00:00:00:00:A0"}))
}
