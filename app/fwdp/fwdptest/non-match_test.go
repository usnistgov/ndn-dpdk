package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestDataWrongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interestB1 := ndntestutil.MakeInterest("/B/1")
	ndntestutil.SetPitToken(interestB1, 0x0290dd7089e9d790)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	dataB2 := ndntestutil.MakeData("/B/2", time.Second) // name does not match
	ndntestutil.CopyPitToken(dataB2, face2.TxInterests[0])
	face2.Rx(dataB2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)
}

func TestDataZeroFreshnessPeriod(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interestB1 := ndntestutil.MakeInterest("/B/1", ndn.MustBeFreshFlag)
	ndntestutil.SetPitToken(interestB1, 0x7ec988011afdf50b)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	dataB1 := ndntestutil.MakeData("/B/1") // no FreshnessPeriod
	ndntestutil.CopyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)
}
