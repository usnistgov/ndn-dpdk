package socketface

import "C"
import (
	"encoding/hex"
	"fmt"
	"net"

	"ndn-dpdk/ndn"
)

type datagramImpl struct{}

func (impl datagramImpl) RxLoop(face *SocketFace) {
	buf := make([]byte, face.rxMp.GetDataroom())
	for {
		nOctets, e := face.conn.Read(buf)
		if e != nil {
			if e2, ok := e.(net.Error); ok && !e2.Temporary() {
				face.logger.Printf("RX socket failed: %v", e)
				return
			}
			face.logger.Printf("RX socket error: %v", e)
			continue
		}

		mbuf, e := face.rxMp.Alloc()
		if e != nil {
			face.logger.Printf("RX alloc error: %v", e)
			continue
		}

		pkt := mbuf.AsPacket()
		seg0 := pkt.GetFirstSegment()
		seg0.SetHeadroom(0)
		seg0.AppendOctets(buf[:nOctets])

		// TODO parse and deliver packets
		face.logger.Printf("RX %d octets %v.., parsing not implemented",
			nOctets, hex.EncodeToString(buf[:16]))
		mbuf.Close()
	}
}

func (impl datagramImpl) TxLoop(face *SocketFace) {
	for {
		select {
		case pkt := <-face.txQueue:
			impl.send(face, pkt)
		case <-face.txQuit:
			return
		}
	}
}

func (impl datagramImpl) send(face *SocketFace, pkt ndn.Packet) {
	if pkt.GetNetType() == ndn.NdnPktType_Nack {
		panic("Nack sending not implemented")
	}

	buf := make([]byte, pkt.Len())
	pkt.ReadTo(0, buf)
	_, e := face.conn.Write(buf)
	if e != nil {
		panic(fmt.Sprintf("conn.Write error %v", e))
	}
}
