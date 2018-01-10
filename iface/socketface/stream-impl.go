package socketface

import "C"
import (
	"net"

	"ndn-dpdk/dpdk"
)

type streamImpl struct {
	face *SocketFace
}

func newStreamImpl(face *SocketFace, conn net.Conn) *streamImpl {
	impl := new(streamImpl)
	impl.face = face
	return impl
}

func (impl *streamImpl) RxLoop() {
	face := impl.face
	face.logger.Print("RxLoop is not implemented, nothing will be received")
	<-face.rxQuit
}

func (impl *streamImpl) Send(pkt dpdk.Packet) error {
	for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
		buf := C.GoBytes(seg.GetData(), C.int(seg.Len()))
		_, e := impl.face.conn.Write(buf)
		if e != nil {
			return e
		}
	}
	return nil
}
