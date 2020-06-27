package socketface

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type streamImpl struct{}

func (streamImpl) RxLoop(face *SocketFace) {
	buffer := make([]byte, face.cfg.RxBufferLength)
	nAvail := 0
	for {
		nRead, e := face.GetConn().Read(buffer[nAvail:])
		if e != nil {
			if face.handleError(e) {
				return
			}
			nAvail = 0 // discard partial packet after the socket has been redialed
			continue
		}
		nAvail += nRead

		// parse and post packets
		d := tlv.Decoder(buffer[:nAvail])
		elements := d.Elements()
		if len(elements) == 0 {
			continue
		}

		for _, de := range elements {
			var packet ndn.Packet
			e := de.Unmarshal(&packet)
			if e != nil { // ignore decoding error
				continue
			}
			face.rx <- &packet
		}

		// move remaining portion to the front
		buffer = make([]byte, face.cfg.RxBufferLength)
		nAvail = copy(buffer, d.Rest())
	}
}

type tcpImpl struct {
	streamImpl
	noLocalAddrDialer
	localAddrRedialer
}

type unixImpl struct {
	streamImpl
	noLocalAddrDialer
	noLocalAddrRedialer
}

func init() {
	var tcp tcpImpl
	implByNetwork["tcp"] = tcp
	implByNetwork["tcp4"] = tcp
	implByNetwork["tcp6"] = tcp
	implByNetwork["unix"] = unixImpl{}
}
