package socketface

import "C"
import (
	"fmt"

	"ndn-dpdk/ndn"
)

type streamImpl struct{}

func (impl streamImpl) RxLoop(face *SocketFace) {
	panic("not implemented")
}

func (impl streamImpl) TxLoop(face *SocketFace) {
	for {
		select {
		case pkt := <-face.txQueue:
			impl.send(face, pkt)
		case <-face.txQuit:
			return
		}
	}
}

func (impl streamImpl) send(face *SocketFace, pkt ndn.Packet) {
	if pkt.GetNetType() == ndn.NdnPktType_Nack {
		panic("Nack sending not implemented")
	}

	for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
		buf := C.GoBytes(seg.GetData(), C.int(seg.Len()))
		_, e := face.conn.Write(buf)
		if e != nil {
			panic(fmt.Sprintf("conn.Write error %v", e))
		}
	}
}
