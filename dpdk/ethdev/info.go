package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/zyedidia/generic"
)

// Driver names.
const (
	DriverAfPacket = "net_af_packet"
	DriverXDP      = "net_af_xdp"
	DriverMemif    = "net_memif"
	DriverRing     = "net_ring"
)

const (
	txOffloadMultiSegs = C.RTE_ETH_TX_OFFLOAD_MULTI_SEGS
	txOffloadChecksum  = C.RTE_ETH_TX_OFFLOAD_IPV4_CKSUM | C.RTE_ETH_TX_OFFLOAD_UDP_CKSUM
)

// DevInfo provides contextual information of an Ethernet port.
type DevInfo struct {
	DevInfoC
}

// DriverName returns DPDK net driver name.
func (info DevInfo) DriverName() string {
	return C.GoString((*C.char)(unsafe.Pointer(info.Driver_name)))
}

// IsVDev determines whether the driver is a virtual device.
func (info DevInfo) IsVDev() bool {
	switch info.DriverName() {
	case DriverAfPacket, DriverXDP, DriverMemif, DriverRing:
		return true
	}
	return false
}

// canIgnoreSetMTUError determines whether set MTU error should be ignored.
func (info DevInfo) canIgnoreSetMTUError() bool {
	switch info.DriverName() {
	case DriverMemif, DriverRing:
		return true
	}
	return false
}

// canIgnorePromiscError determines whether enable/disable promiscuous mode error should be ignored.
func (info DevInfo) canIgnorePromiscError() bool {
	switch info.DriverName() {
	case DriverMemif, DriverRing:
		return true
	}
	return false
}

// HasTxMultiSegOffload determines whether device can transmit multi-segment packets.
func (info DevInfo) HasTxMultiSegOffload() bool {
	if info.Tx_offload_capa&txOffloadMultiSegs == txOffloadMultiSegs {
		return true
	}

	switch info.DriverName() { // some drivers support multi-segment TX but do not advertise it
	case DriverRing:
		return true
	}
	return false
}

// HasTxChecksumOffload determines whether device can compute IPv4 and UDP checksum upon transmission.
func (info DevInfo) HasTxChecksumOffload() bool {
	return info.Tx_offload_capa&txOffloadChecksum == txOffloadChecksum
}

// MarshalJSON implements json.Marshaler interface.
func (info DevInfo) MarshalJSON() ([]byte, error) {
	var m map[string]any
	if e := jsonhelper.Roundtrip(info.DevInfoC, &m); e != nil {
		return nil, e
	}
	typ, val := reflect.TypeOf(info), []reflect.Value{reflect.ValueOf(info)}
	for i, n := 0, typ.NumMethod(); i < n; i++ {
		method := typ.Method(i)
		if method.IsExported() && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 {
			m[method.Name] = method.Func.Call(val)[0].Interface()
		}
	}
	return json.Marshal(m)
}

// adjustQueueCapacity adjust RX/TX queue capacity to satisfy driver requirements.
func (lim DescLim) adjustQueueCapacity(capacity int) int {
	capacity -= capacity % int(lim.Align)
	return generic.Clamp(capacity, int(lim.Min), int(lim.Max))
}

func (stats Stats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		stats.Ipackets, stats.Ibytes, stats.Imissed, stats.Ierrors, stats.Rx_nombuf, stats.Opackets, stats.Obytes, stats.Oerrors)
}
