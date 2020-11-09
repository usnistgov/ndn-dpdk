package ealconfig_test

import (
	"flag"
	"testing"

	"github.com/soh335/sliceflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

type testHwInfo struct{}

func (testHwInfo) Cores() (list []ealconfig.CoreInfo) {
	for coreID := 0; coreID < 32; coreID++ {
		list = append(list, ealconfig.CoreInfo{
			ID:          coreID,
			NumaSocket:  coreID % 8,
			HyperThread: coreID < 8 || coreID >= 24,
		})
	}
	return list
}

func makeBaseConfig() (cfg ealconfig.Config) {
	cfg.LCoreFlags = "--skip-lcore"
	cfg.MemFlags = "--skip-mem"
	cfg.DeviceFlags = "--skip-device"
	return cfg
}

func makeBaseFlagSet() (fset *flag.FlagSet) {
	fset = flag.NewFlagSet("", flag.PanicOnError)
	fset.Bool("skip-lcore", false, "")
	fset.Bool("skip-mem", false, "")
	fset.Bool("skip-device", false, "")
	return fset
}

func parseExtraFlags(args []string) (a, b string) {
	fset := makeBaseFlagSet()
	fset.StringVar(&a, "flag-a", "", "")
	fset.StringVar(&b, "flag-b", "", "")
	fset.Parse(args)
	return
}

func TestReplaceFlags(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = "--flag-a value-a"
	cfg.Flags = "--flag-b value-b"

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	a, b := parseExtraFlags(args)
	assert.Equal("", a)
	assert.Equal("value-b", b)
}

func TestExtraFlags(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = "--flag-a value-a"
	cfg.ExtraFlags = "--flag-b value-b"

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	a, b := parseExtraFlags(args)
	assert.Equal("value-a", a)
	assert.Equal("value-b", b)
}

func parseLCoreFlags(args []string) (l, lcores string) {
	fset := makeBaseFlagSet()
	fset.StringVar(&l, "l", "", "")
	fset.StringVar(&lcores, "lcores", "", "")
	fset.Parse(args)
	return
}

func TestLCoreCores(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = ""
	cfg.Cores = []int{0, 1, 4, 7, 32}

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	l, lcores := parseLCoreFlags(args)
	commaSetEquals(assert, "0,1,4,7", l)
	assert.Equal("", lcores)
}

func TestLCoreFewerCores(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = ""
	cfg.Cores = []int{0, 1, 4, 7, 32}

	req := ealconfig.Request{
		MinLCores: 6,
	}

	args, e := cfg.Args(req, testHwInfo{})
	require.NoError(e)
	l, lcores := parseLCoreFlags(args)
	assert.Equal("", l)
	assert.Equal("(0-5)@(0,1,4,7)", lcores)
}

func TestLCoreNoCores(t *testing.T) {
	assert, _ := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = ""
	cfg.Cores = []int{32}

	_, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	assert.Error(e)
}

func TestLCorePerNuma(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.LCoreFlags = ""
	cfg.CoresPerNuma = map[int]int{
		// 0: 8,16,0,24
		1: 2,  // 9,17
		2: 4,  // 10,18,2,26
		3: 5,  // 11,19,3,27
		4: 0,  // none
		5: -3, // 13
		6: -4, // none
		7: -5, // none
		8: 1,  // non-existent socket
	}

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	l, lcores := parseLCoreFlags(args)
	commaSetEquals(assert, "8,16,0,24,9,17,10,18,2,26,11,19,3,27,13", l)
	assert.Equal("", lcores)
}

func parseMemFlags(args []string) (n, socketLimit, filePrefix string, hugeUnlink bool) {
	fset := makeBaseFlagSet()
	fset.StringVar(&n, "n", "", "")
	fset.StringVar(&socketLimit, "socket-limit", "", "")
	fset.StringVar(&filePrefix, "file-prefix", "", "")
	fset.BoolVar(&hugeUnlink, "huge-unlink", false, "")
	fset.Parse(args)
	return
}

func TestMemoryEmpty(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.MemFlags = ""

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	n, socketLimit, filePrefix, hugeUnlink := parseMemFlags(args)
	assert.Equal("", n)
	assert.Equal("", socketLimit)
	assert.Equal("", filePrefix)
	assert.True(hugeUnlink)
}

func TestMemoryAll(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.MemFlags = ""
	cfg.MemChannels = 2
	cfg.MemPerNuma = map[int]int{
		1: 1024,
		2: 2048,
		3: 4096,
		6: 0,    // 1
		8: 8192, // non-existent
	}
	cfg.FilePrefix = "ealconfigtest"
	cfg.DisableHugeUnlink = true

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	n, socketLimit, filePrefix, hugeUnlink := parseMemFlags(args)
	assert.Equal("2", n)
	assert.Equal("0,1024,2048,4096,0,0,1,0", socketLimit)
	assert.Equal("ealconfigtest", filePrefix)
	assert.False(hugeUnlink)
}

func parseDeviceFlags(args []string) (w, vdev []string, noPci bool) {
	fset := makeBaseFlagSet()
	sliceflag.StringVar(fset, &w, "w", nil, "")
	sliceflag.StringVar(fset, &vdev, "vdev", nil, "")
	fset.BoolVar(&noPci, "no-pci", false, "")
	fset.Parse(args)
	return
}

func TestDeviceEmpty(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.DeviceFlags = ""

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	w, vdev, noPci := parseDeviceFlags(args)
	assert.Len(w, 0)
	assert.Len(vdev, 0)
	assert.True(noPci)
}

func TestDeviceSome(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.DeviceFlags = ""
	cfg.PciDevices = []ealconfig.PciAddress{
		ealconfig.MustParsePciAddress("02:00.0"),
		ealconfig.MustParsePciAddress("0A:00.0"),
	}
	cfg.VirtualDevices = []string{
		"net_af_packet1,iface=eth1",
	}

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	w, vdev, noPci := parseDeviceFlags(args)
	assert.Equal([]string{"0000:02:00.0", "0000:0a:00.0"}, w)
	assert.Equal([]string{"net_af_packet1,iface=eth1"}, vdev)
	assert.False(noPci)
}

func TestDeviceAll(t *testing.T) {
	assert, require := makeAR(t)

	cfg := makeBaseConfig()
	cfg.DeviceFlags = ""
	cfg.AllPciDevices = true
	cfg.VirtualDevices = []string{
		"net_af_packet1,iface=eth1",
	}

	args, e := cfg.Args(ealconfig.Request{}, testHwInfo{})
	require.NoError(e)
	w, vdev, noPci := parseDeviceFlags(args)
	assert.Len(w, 0)
	assert.Equal([]string{"net_af_packet1,iface=eth1"}, vdev)
	assert.False(noPci)
}
