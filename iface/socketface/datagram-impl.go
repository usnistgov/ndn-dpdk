package socketface

import "C"
import (
	"fmt"

	"ndn-dpdk/ndn"
)

type datagramImpl struct{}

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
