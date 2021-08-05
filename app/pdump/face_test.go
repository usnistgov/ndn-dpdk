package pdump_test

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
	defer ealthread.AllocClear()
	assert, require := makeAR(t)

	filename, del := testenv.TempName()
	defer del()

	w, e := pdump.NewWriter(pdump.WriterConfig{
		Filename:     filename,
		MaxSize:      1 << 20,
		RingCapacity: 4096,
	})
	require.NoError(e)
	require.NoError(ealthread.AllocLaunch(w))

	face := intface.MustNew()
	go func() {
		for pkt := range face.Rx {
			if pkt.Interest != nil {
				face.Tx <- ndn.MakeData(pkt.Interest)
			}
		}
	}()

	dumpRx, e := pdump.DumpFace(face.D, w, pdump.FaceConfig{
		Dir: pdump.DirIncoming,
		Names: []pdump.NameFilterEntry{
			{Name: ndn.ParseName("/"), SampleRate: 0.8},
		},
	})
	require.NoError(e)
	dumpTx, e := pdump.DumpFace(face.D, w, pdump.FaceConfig{
		Dir: pdump.DirOutgoing,
		Names: []pdump.NameFilterEntry{
			{Name: ndn.ParseName("/"), SampleRate: 0.3},
		},
	})
	require.NoError(e)

	const nBursts, nBurstSize = 128, 16
	for i := 0; i < nBursts; i++ {
		pkts := make([]*ndni.Packet, nBurstSize)
		for j := range pkts {
			pkts[j] = makeInterest(fmt.Sprintf("/%d/%d", i, j))
		}
		iface.TxBurst(face.ID, pkts)
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	assert.NoError(dumpRx.Close())
	assert.NoError(dumpTx.Close())
	assert.NoError(w.Close())
	face.D.Close()

	f, e := os.Open(filename)
	require.NoError(e)
	defer f.Close()
	r, e := pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	require.NoError(e)

	nRxData, nTxInterests := 0, 0
	var sll layers.LinuxSLL
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeLinuxSLL, &sll)
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
					assert.NotNil(npkt.Interest)
					nTxInterests++
				default:
					assert.Fail("unexpected sll.PacketType")
				}
			}
		}
	}
	assert.InEpsilon(nBursts*nBurstSize*0.8, nRxData, 0.5)
	assert.InEpsilon(nBursts*nBurstSize*0.3, nTxInterests, 0.5)

	if save := os.Getenv("PDUMPTEST_SAVE"); save != "" {
		os.Rename(filename, save)
	}
}
