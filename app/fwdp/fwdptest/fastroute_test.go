package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestFastroute(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	face4 := fixture.CreateFace()
	fixture.SetFibEntry("/A/B", "fastroute", face1.GetFaceId(), face2.GetFaceId(), face3.GetFaceId())

	// multicast first Interest
	interest1 := ndntestutil.MakeInterest("/A/B/1")
	face4.Rx(interest1)
	time.Sleep(10 * time.Millisecond)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	require.Len(face3.TxInterests, 1)

	// face3 replies Data
	data1 := ndntestutil.MakeData("/A/B/1")
	ndntestutil.CopyPitToken(data1, face3.TxInterests[0])
	face3.Rx(data1)
	time.Sleep(10 * time.Millisecond)

	// unicast subsequent Interest
	interest2 := ndntestutil.MakeInterest("/A/B/2")
	face4.Rx(interest2)
	time.Sleep(10 * time.Millisecond)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 2)

	// unicast subsequent Interest
	interest3 := ndntestutil.MakeInterest("/A/B/3")
	face4.Rx(interest3)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 3)

	// face3 fails
	face3.SetDown(true)

	// multicast next Interest because face failed
	interest4 := ndntestutil.MakeInterest("/A/B/4")
	face4.Rx(interest4)
	time.Sleep(10 * time.Millisecond)
	assert.Len(face1.TxInterests, 2)
	assert.Len(face2.TxInterests, 2)
	assert.Len(face3.TxInterests, 3)
}
