package pdumptest

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/app/pdump"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethringdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestEthPortUnmatched(t *testing.T) {
	assert, require := makeAR(t)
	filename := filepath.Join(t.TempDir(), "pdump.pcapng")

	w, e := pdump.NewWriter(pdump.WriterConfig{
		Filename:     filename,
		MaxSize:      1 << 22,
		RingCapacity: 4096,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))
	defer ealthread.AllocFree(w.LCore())

	pair, e := ethringdev.NewPair(ethringdev.PairConfig{
		RxPool: mbuftestenv.DirectMempool(),
	})
	require.NoError(e)
	portA, e := ethport.New(ethport.Config{EthDev: pair.PortA})
	require.NoError(e)
	defer portA.Close()
	portB, e := ethport.New(ethport.Config{EthDev: pair.PortB})
	require.NoError(e)
	defer portB.Close()

	locA0 := ethface.EtherLocator{}
	locA0.EthDev = portA.EthDev()
	locA0.Local.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:A0")
	locA0.Remote.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:B0")
	faceA0, e := locA0.CreateFace()
	require.NoError(e)
	defer faceA0.Close()

	locB0 := ethface.EtherLocator{}
	locB0.EthDev = portB.EthDev()
	locB0.Local.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:B0")
	locB0.Remote.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:A0")
	faceB0, e := locB0.CreateFace()
	require.NoError(e)
	defer faceB0.Close()

	locB1 := ethface.EtherLocator{}
	locB1.EthDev = portB.EthDev()
	locB1.Local.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:B1")
	locB1.Remote.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:A1")
	faceB1, e := locB1.CreateFace()
	require.NoError(e)
	defer faceB1.Close()

	locB2 := ethface.UDPLocator{}
	locB2.EthDev = portB.EthDev()
	locB2.Local.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:B0")
	locB2.LocalIP, locB2.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locB2.Remote.HardwareAddr, _ = net.ParseMAC("02:00:00:00:00:A0")
	locB2.RemoteIP, locB2.RemoteUDP = netip.MustParseAddr("192.168.1.1"), 6363
	faceB2, e := locB2.CreateFace()
	require.NoError(e)
	defer faceB2.Close()

	dump, e := pdump.NewEthPortSource(pdump.EthPortConfig{
		Writer: w,
		Port:   portA,
		Grab:   pdump.EthGrabRxUnmatched,
	})
	require.NoError(e)

	const nBursts, nBurstSize = 512, 16
	for i := range nBursts {
		for k, face := range []iface.Face{faceB0, faceB1, faceB2} {
			pkts := make([]*ndni.Packet, nBurstSize)
			for j := range pkts {
				pkts[j] = makeInterest(fmt.Sprintf("/%d/%d/%d", i, j, k))
			}
			iface.TxBurst(face.ID(), pkts)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	assert.NoError(dump.Close())
	time.Sleep(100 * time.Millisecond)
	assert.NoError(w.Close())

	f, e := os.Open(filename)
	require.NoError(e)
	defer f.Close()
	r, e := pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	require.NoError(e)

	count := 0
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var udp layers.UDP
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &udp)
	parser.IgnoreUnsupported = true
	decoded := []gopacket.LayerType{}
	for {
		pkt, _, e := r.ReadPacketData()
		if errors.Is(e, io.EOF) {
			break
		}
		if !assert.NoError(parser.DecodeLayers(pkt, &decoded)) {
			continue
		}
		switch len(decoded) {
		case 1:
			assert.Equal(layers.LayerTypeEthernet, decoded[0])
			assert.Equal(locB1.Remote.HardwareAddr, eth.DstMAC)
			assert.Equal(locB1.Local.HardwareAddr, eth.SrcMAC)
		case 3:
			assert.Equal(layers.LayerTypeEthernet, decoded[0])
			assert.Equal(layers.LayerTypeIPv4, decoded[1])
			assert.Equal(layers.LayerTypeUDP, decoded[2])
			assert.Equal(locB2.Remote.HardwareAddr, eth.DstMAC)
			assert.Equal(locB2.Local.HardwareAddr, eth.SrcMAC)
		default:
			assert.Fail("unexpected decoded layers", decoded)
		}
		count++
	}
	assert.InEpsilon(nBursts*nBurstSize*2, count, 0.4)

	if save := os.Getenv("PDUMPTEST_SAVE"); save != "" {
		os.Rename(filename, save)
	}
}
