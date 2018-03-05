package socketface

import "C"
import (
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
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
	buf := make(ndn.TlvBytes, face.rxMp.GetDataroom())
	nAvail := 0
	for {
		nRead, e := face.conn.Read(buf[nAvail:])
		if e != nil {
			if face.handleError("RX", e) {
				return
			}
			continue
		}
		nAvail += nRead

		// parse and post packets
		offset := 0
		for {
			n := impl.postPacket(buf[offset:nAvail])
			if n == 0 {
				break
			}
			offset += n
		}

		// move remaining portion to the front
		for i := offset; i < nAvail; i++ {
			buf[i-offset] = buf[i]
		}
		nAvail -= offset

		select {
		case <-face.rxQuit:
			return
		default:
		}
	}
}

func (impl *streamImpl) postPacket(buf ndn.TlvBytes) (n int) {
	face := impl.face

	element, _ := buf.ExtractElement()
	if element == nil {
		return 0
	}

	mbuf, e := face.rxMp.Alloc()
	if e != nil {
		face.logger.Printf("RX alloc error: %v", e)
		return n
	}

	pkt := mbuf.AsPacket()
	pkt.SetPort(uint16(impl.face.GetFaceId()))
	pkt.SetTimestamp(dpdk.TscNow())
	seg0 := pkt.GetFirstSegment()
	seg0.SetHeadroom(0)
	seg0.Append([]byte(element))

	select {
	case face.rxQueue <- pkt:
	default:
		pkt.Close()
		face.rxReportCongestion()
	}

	return len(element)
}

func (impl *streamImpl) Send(pkt dpdk.Packet) error {
	for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
		buf := seg.AsByteSlice()
		_, e := impl.face.conn.Write(buf)
		if e != nil {
			return e
		}
	}
	return nil
}
