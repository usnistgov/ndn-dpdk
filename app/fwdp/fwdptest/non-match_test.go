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

	interest := ndntestutil.MakeInterest("/B/1")
	face1.Rx(interest)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	data := ndntestutil.MakeData("/B/2", time.Second) // name does not match
	ndntestutil.CopyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)
}

func TestDataLongerName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := ndntestutil.MakeInterest("/B/1") // no CanBePrefix
	face1.Rx(interest)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	data := ndntestutil.MakeData("/B/1/Z", time.Second) // name has suffix
	ndntestutil.CopyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
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

	interest := ndntestutil.MakeInterest("/B/1", ndn.MustBeFreshFlag) // has MustBeFresh
	face1.Rx(interest)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	data := ndntestutil.MakeData("/B/1") // no FreshnessPeriod
	ndntestutil.CopyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)
}
