package fib_test

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestReplica1(t *testing.T) {
	assert, require := makeAR(t)

	var th0, th1 fibtestenv.LookupThread
	f, e := fib.New(fibdef.Config{
		Capacity:   1023,
		StartDepth: 2,
	}, []fib.LookupThread{&th0, &th1})
	require.NoError(e)
	defer f.Close()

	assert.Equal(f.Replica(eal.NumaSocket{}).Ptr(), th0.Replica)
	assert.Equal(th0.Replica, th1.Replica)
	assert.NotEqual(th0.Index, th1.Index)
	assert.Less(th0.Index, 2)
	assert.Less(th1.Index, 2)
}

func TestReplica2(t *testing.T) {
	if len(eal.Sockets) < 2 {
		fmt.Println("skipping TestReplica2: only one NUMA socket")
		return
	}
	assert, require := makeAR(t)

	var th0, th1 fibtestenv.LookupThread
	th0.Socket = eal.Sockets[0]
	th1.Socket = eal.Sockets[1]
	f, e := fib.New(fibdef.Config{
		Capacity:   1023,
		StartDepth: 2,
	}, []fib.LookupThread{&th0, &th1})
	require.NoError(e)
	defer f.Close()

	assert.Equal(f.Replica(eal.Sockets[0]).Ptr(), th0.Replica)
	assert.Equal(f.Replica(eal.Sockets[1]).Ptr(), th1.Replica)
	assert.NotEqual(th0.Replica, th1.Replica)
	assert.Equal(0, th0.Index)
	assert.Equal(0, th1.Index)
}

func TestLpm(t *testing.T) {
	assert, require := makeAR(t)

	var th0, th1 fibtestenv.LookupThread
	if len(eal.Sockets) >= 2 {
		th0.Socket = eal.Sockets[0]
		th1.Socket = eal.Sockets[1]
	}
	f, e := fib.New(fibdef.Config{
		Capacity:   1023,
		StartDepth: 2,
	}, []fib.LookupThread{&th0, &th1})
	require.NoError(e)
	defer f.Close()

	checkEntryNames := func(expectedInput ...string) {
		expected := make([]string, len(expectedInput))
		for i, uri := range expectedInput {
			expected[i] = ndn.ParseName(uri).String()
		}

		entries := f.List()
		actual := make([]string, len(entries))
		for i, entry := range entries {
			actual[i] = entry.Name.String()
		}

		assert.ElementsMatch(expected, actual)
	}

	lpm := func(name string) iface.ID {
		n := ndn.ParseName(name)
		entryR := f.Replica(th0.Socket).Lpm(n)
		if entryR == nil {
			if th1.Socket != th0.Socket {
				assert.Nil(f.Replica(th1.Socket).Lpm(n), "%s", name)
			}
			return 0
		}
		entry := entryR.Read()

		if th1.Socket != th0.Socket {
			entryR1 := f.Replica(th1.Socket).Lpm(n)
			if assert.NotNil(entryR1, "%s", name) {
				entry1 := entryR1.Read()
				nameEqual(assert, entry, entry1)
				assert.Equal(entry.Nexthops, entry1.Nexthops)
			}
		}
		return entry.Nexthops[0]
	}
	checkLpms := func(expected ...iface.ID) {
		assert.Equal(expected, []iface.ID{
			lpm("/"),
			lpm("/A"),
			lpm("/AB"),
			lpm("/A/B"),
			lpm("/A/B/C"),
			lpm("/A/B/C/D"),
			lpm("/A/B/CD"),
			lpm("/E/F/G/H"),
			lpm("/E/F/I"),
			lpm("/J"),
			lpm("/J/K"),
			lpm("/J/K/L"),
			lpm("/J/K/M/N/O"),
			lpm("/U/V/W/X/Y/Z"),
			lpm("/U/V/W"),
			lpm("/U/V"),
			lpm("/U"),
		})
	}

	f.Insert(makeEntry("/", nil, 5000))
	f.Insert(makeEntry("/A", nil, 5100))
	f.Insert(makeEntry("/A/B/C", nil, 5101))   // insert virtual /A/B
	f.Insert(makeEntry("/E/F/G/H", nil, 5200)) // insert virtual /E/F
	f.Insert(makeEntry("/E/F/I", nil, 5201))   // don't update virtual /E/F
	f.Insert(makeEntry("/J/K", nil, 5300))
	f.Insert(makeEntry("/J/K/L", nil, 5301))   // insert virtual /J/K
	f.Insert(makeEntry("/J/K/M/N", nil, 5302)) // update virtual /J/K
	f.Insert(makeEntry("/U/V/W/X", nil, 5400)) // insert virtual /U/V
	f.Insert(makeEntry("/U/V/W", nil, 5401))   // don't update virtual /U/V
	f.Insert(makeEntry("/U/V", nil, 5402))
	f.Insert(makeEntry("/U", nil, 5403))

	assert.Equal(12, f.Len())
	checkEntryNames("/", "/A", "/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(5000, 5100, 5000, 5100, 5101, 5101, 5100, 5200, 5201, 5000, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/")))
	assert.Equal(11, f.Len())
	checkEntryNames("/A", "/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 5100, 0, 5100, 5101, 5101, 5100, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/A")))
	assert.Equal(10, f.Len())
	checkEntryNames("/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 5101, 5101, 0, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/A/B/C"))) // erase virtual /A/B
	assert.Equal(9, f.Len())
	checkEntryNames("/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/E/F/G/H"))) // update virtual /E/F
	assert.Equal(8, f.Len())
	checkEntryNames("/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/E/F/I"))) // erase virtual /E/F
	assert.Equal(7, f.Len())
	checkEntryNames("/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/J/K")))
	assert.Equal(6, f.Len())
	checkEntryNames("/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5301, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/J/K/L"))) // don't update virtual /J/K
	assert.Equal(5, f.Len())
	checkEntryNames("/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5302, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/J/K/M/N"))) // erase virtual /J/K
	assert.Equal(4, f.Len())
	checkEntryNames("/U", "/U/V", "/U/V/W", "/U/V/W/X")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5400, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/U/V/W/X"))) // update virtual /U/V
	assert.Equal(3, f.Len())
	checkEntryNames("/U", "/U/V", "/U/V/W")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5401, 5401, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/U/V/W"))) // erase virtual /U/V
	assert.Equal(2, f.Len())
	checkEntryNames("/U", "/U/V")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5402, 5402, 5402, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/U/V")))
	assert.Equal(1, f.Len())
	checkEntryNames("/U")
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5403, 5403, 5403, 5403)

	assert.NoError(f.Erase(ndn.ParseName("/U")))
	assert.Equal(0, f.Len())
	checkEntryNames()
	checkLpms(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}
