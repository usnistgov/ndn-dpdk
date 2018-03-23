package cstest

import (
	"testing"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn/ndntestutil"
)

func createCs() cs.Cs {
	cfg := pcct.Config{
		Id:         "TestPcct",
		MaxEntries: 255,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}

	pcct, e := pcct.New(cfg)
	if e != nil {
		panic(e)
	}
	return cs.Cs{pcct}
}

func pitFromCs(cs cs.Cs) pit.Pit {
	return pit.Pit{cs.Pcct}
}

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	cs := createCs()
	defer cs.Pcct.Close()
	pit := pitFromCs(cs)
	mp := cs.GetMempool()
	assert.Zero(pit.Len())
	assert.Zero(mp.CountInUse())

	interest := ndntestutil.MakeInterest("/A/B")
	pitEntry, csEntry := pit.Insert(interest)
	assert.Nil(csEntry)
	require.NotNil(pitEntry)

	data := ndntestutil.MakeData("/A/B")
	ndntestutil.SetPitToken(data, pitEntry.GetToken())
	pitFound := pit.FindByData(data)
	assert.Equal(1, pitFound.Len())

	assert.Zero(cs.Len())
	cs.Insert(data, pitFound)
	assert.Equal(1, cs.Len())
	assert.Len(cs.List(), 1)
	assert.Zero(pit.Len())
	assert.Equal(1, mp.CountInUse())

	pitEntry, csEntry = pit.Insert(interest)
	assert.Nil(pitEntry)
	require.NotNil(csEntry)

	cs.Erase(*csEntry)
	assert.Zero(cs.Len())
	assert.Zero(mp.CountInUse())
	assert.Len(cs.List(), 0)
}
