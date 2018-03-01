package socketface_test

import (
	"net"
	"testing"

	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/socketface"
)

func TestDatagram(t *testing.T) {
	_, require := makeAR(t)

	addrA := net.UDPAddr{net.ParseIP("127.0.0.1"), 7001, ""}
	addrB := net.UDPAddr{net.ParseIP("127.0.0.1"), 7002, ""}
	connA, e := net.DialUDP("udp", &addrB, &addrA)
	require.NoError(e)
	connB, e := net.DialUDP("udp", &addrA, &addrB)
	require.NoError(e)

	faceA := socketface.New(connA, socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	faceB := socketface.New(connB, socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	defer faceA.Close()
	defer faceB.Close()

	fixture := ifacetestfixture.New(t, faceA, socketface.NewRxGroup(faceA), faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}
