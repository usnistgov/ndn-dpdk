package socketface

import "C"
import (
	"fmt"
	"log"
	"net"

	"ndn-dpdk/ndn"
)

type StreamFace struct {
	baseFace
	failed bool
}

func NewStreamFace(conn net.Conn) (face *StreamFace) {
	face = new(StreamFace)
	// TODO allocate FaceId
	face.conn = conn
	return face
}

func (face *StreamFace) String() string {
	return fmt.Sprintf("%d %s %v <=> %v", face.id, face.conn.LocalAddr().Network(),
		face.conn.LocalAddr(), face.conn.RemoteAddr())
}

func (face *StreamFace) RxBurst(pkts []ndn.Packet) int {
	panic("not implemented")
}

func (face *StreamFace) TxBurst(pkts []ndn.Packet) {
	// TODO NDNLP encoding

	if face.failed {
		log.Print("face %d has failed", face.id)
		return
	}

	for _, pkt := range pkts {
		for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
			data := C.GoBytes(seg.GetData(), C.int(seg.Len()))
			_, e := face.conn.Write(data) // TODO non-blocking
			if e != nil {
				log.Print("face %d is failing %v", face.id, e)
				face.failed = true // TODO better failure handling
			}
		}
	}
}
