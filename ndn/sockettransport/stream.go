package sockettransport

import (
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type streamImpl struct{}

func (streamImpl) RxLoop(tr *Transport) {
	buffer := make([]byte, tr.cfg.RxBufferLength)
	nAvail := 0
	for {
		nRead, e := tr.GetConn().Read(buffer[nAvail:])
		if e != nil {
			if tr.handleError(e) {
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
			tr.rx <- de.Wire
		}

		// copy remaining portion to a new buffer
		// don't reuse buffer because the packets passed to tr.rx is still referencing it
		buffer = make([]byte, tr.cfg.RxBufferLength)
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
