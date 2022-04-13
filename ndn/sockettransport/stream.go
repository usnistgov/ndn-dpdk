package sockettransport

import "github.com/usnistgov/ndn-dpdk/ndn/tlv"

func streamDecode(received, buf []byte) (rest []byte, n int) {
	if len(received) > 0 {
		d := tlv.DecodingBuffer(received)
		if de, e := d.Element(); e == nil {
			return de.After, copy(buf, de.Wire)
		}
	}
	return received, 0
}

type streamReader struct{}

func (streamReader) Read(tr *transport, trc *trConn, buf []byte) (n int, e error) {
	received, _ := trc.rx.([]byte)
	received, n = streamDecode(received, buf)
	if n > 0 {
		trc.rx = received
		return n, nil
	}

	if mtu := tr.MTU(); cap(received) < mtu {
		received = append(make([]byte, 0, 2*mtu), received...)
	}
	r, e := trc.conn.Read(received[len(received):cap(received)])
	if e != nil {
		return 0, e
	}
	received = received[:len(received)+r]

	received, n = streamDecode(received, buf)
	trc.rx = received
	return n, nil
}

type tcpImpl struct {
	noLocalAddrDialer
	localAddrRedialer
	streamReader
}

type unixImpl struct {
	noLocalAddrDialer
	noLocalAddrRedialer
	streamReader
}

func init() {
	implByNetwork["tcp"] = tcpImpl{}
	implByNetwork["tcp4"] = tcpImpl{}
	implByNetwork["tcp6"] = tcpImpl{}

	implByNetwork["unix"] = unixImpl{}
}
