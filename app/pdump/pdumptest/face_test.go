package pdumptest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/usnistgov/ndn-dpdk/app/pdump"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestFaceRxTx(t *testing.T) {
	assert, require := makeAR(t)

	filename, del := testenv.TempName()
	defer del()
	w, e := pdump.NewWriter(pdump.WriterConfig{
		Filename:     filename,
		MaxSize:      1 << 22,
		RingCapacity: 4096,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))
	defer ealthread.AllocFree(w.LCore())

	face := intface.MustNew()
	go func() {
		for pkt := range face.Rx {
			if pkt.Interest != nil {
				face.Tx <- ndn.MakeData(pkt.Interest)
			}
		}
	}()

	dumpRx, e := pdump.NewFaceSource(pdump.FaceConfig{
		Writer: w,
		Face:   face.D,
		Dir:    pdump.DirIncoming,
		Names: []pdump.NameFilterEntry{
			{Name: ndn.ParseName("/"), SampleProbability: 0.8},
		},
	})
	require.NoError(e)
	dumpTx, e := pdump.NewFaceSource(pdump.FaceConfig{
		Writer: w,
		Face:   face.D,
		Dir:    pdump.DirOutgoing,
		Names: []pdump.NameFilterEntry{
			{Name: ndn.ParseName("/0"), SampleProbability: 0.5},
			{Name: ndn.ParseName("/3"), SampleProbability: 0.2},
		},
	})
	require.NoError(e)

	const nBursts, nBurstSize = 512, 16
	for i := 0; i < nBursts; i++ {
		pkts := make([]*ndni.Packet, nBurstSize)
		for j := range pkts {
			pkts[j] = makeInterest(fmt.Sprintf("/%d/%d/%d", i%4, i, j))
		}
		iface.TxBurst(face.ID, pkts)
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	assert.NoError(dumpRx.Close())
	_ = dumpTx // closing the face should automatically close dumpers
	face.D.Close()
	time.Sleep(100 * time.Millisecond)
	assert.NoError(w.Close())

	f, e := os.Open(filename)
	require.NoError(e)
	defer f.Close()
	r, e := pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	require.NoError(e)

	prefix0, prefix3 := ndn.ParseName("/0"), ndn.ParseName("/3")
	nRxData, nTxInterests0, nTxInterests3 := 0, 0, 0
	var sll layers.LinuxSLL
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeLinuxSLL, &sll)
	parser.IgnoreUnsupported = true
	decoded := []gopacket.LayerType{}
	for {
		pkt, _, e := r.ReadPacketData()
		if errors.Is(e, io.EOF) {
			break
		}
		if assert.NoError(parser.DecodeLayers(pkt, &decoded)) &&
			assert.Len(decoded, 1) &&
			assert.Equal(layers.LayerTypeLinuxSLL, decoded[0]) {
			var npkt ndn.Packet
			if assert.NoError(tlv.Decode(sll.Payload, &npkt)) {
				switch sll.PacketType {
				case layers.LinuxSLLPacketTypeHost:
					assert.NotNil(npkt.Data)
					nRxData++
				case layers.LinuxSLLPacketTypeOutgoing:
					if assert.NotNil(npkt.Interest) {
						if prefix0.IsPrefixOf(npkt.Interest.Name) {
							nTxInterests0++
						} else if assert.True(prefix3.IsPrefixOf(npkt.Interest.Name)) {
							nTxInterests3++
						}
					}
				default:
					assert.Fail("unexpected sll.PacketType")
				}
			}
		}
	}
	assert.InEpsilon(nBursts*nBurstSize*0.8, nRxData, 0.4)
	assert.InEpsilon(nBursts*nBurstSize/4*0.5, nTxInterests0, 0.4)
	assert.InEpsilon(nBursts*nBurstSize/4*0.3, nTxInterests3, 0.4)

	if save := os.Getenv("PDUMPTEST_SAVE"); save != "" {
		os.Rename(filename, save)
	}
}
