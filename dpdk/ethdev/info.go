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

	"github.com/usnistgov/ndn-dpdk/core/cptr"
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

func infoJSON(info, infoC any) ([]byte, error) {
	var m map[string]any
	if e := jsonhelper.Roundtrip(infoC, &m); e != nil {
		return nil, e
	}

	if info != nil {
		typ, val := reflect.TypeOf(info), []reflect.Value{reflect.ValueOf(info)}
		for i, n := 0, typ.NumMethod(); i < n; i++ {
			method := typ.Method(i)
			if method.IsExported() && method.Name != "String" &&
				method.Type.NumIn() == 1 && method.Type.NumOut() == 1 {
				m[method.Name] = method.Func.Call(val)[0].Interface()
			}
		}
	}

	return json.Marshal(jsonhelper.CleanCgoStruct(m))
}

// DevInfo provides contextual information of an Ethernet port.
type DevInfo struct {
	DevInfoC
}

// Driver returns DPDK net driver name.
func (info DevInfo) Driver() string {
	return C.GoString((*C.char)(unsafe.Pointer(info.Driver_name)))
}

// IsVDev determines whether the driver is a virtual device.
func (info DevInfo) IsVDev() bool {
	switch info.Driver() {
	case DriverAfPacket, DriverXDP, DriverMemif, DriverRing:
		return true
	}
	return false
}

// canIgnoreSetMTUError determines whether set MTU error should be ignored.
func (info DevInfo) canIgnoreSetMTUError() bool {
	switch info.Driver() {
	case DriverMemif, DriverRing:
		return true
	}
	return false
}

// canIgnorePromiscError determines whether enable/disable promiscuous mode error should be ignored.
func (info DevInfo) canIgnorePromiscError() bool {
	switch info.Driver() {
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

	switch info.Driver() { // some drivers support multi-segment TX but do not advertise it
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
	return infoJSON(info, info.DevInfoC)
}

// adjustQueueCapacity adjust RX/TX queue capacity to satisfy driver requirements.
func (lim DescLim) adjustQueueCapacity(capacity int) int {
	capacity -= capacity % int(lim.Align)
	return generic.Clamp(capacity, int(lim.Min), int(lim.Max))
}

// Stats contains statistics for an Ethernet port.
type Stats struct {
	StatsBasic
	dev ethDev
}

// X retrieves extended statistics.
func (stats Stats) X() (m map[string]any) {
	n := C.rte_eth_xstats_get_names(stats.dev.cID(), nil, 0)
	if n <= 0 {
		return nil
	}
	names, xstats := make([]C.struct_rte_eth_xstat_name, n), make([]C.struct_rte_eth_xstat, n)
	if res := C.rte_eth_xstats_get_names(stats.dev.cID(), unsafe.SliceData(names), C.unsigned(n)); res != n {
		return nil
	}
	if res := C.rte_eth_xstats_get(stats.dev.cID(), unsafe.SliceData(xstats), C.unsigned(n)); res != n {
		return nil
	}

	m = map[string]any{}
	for i, name := range names {
		m[cptr.GetString(name.name[:])] = uint64(xstats[i].value)
	}
	return m
}

// MarshalJSON implements json.Marshaler interface.
func (stats Stats) MarshalJSON() ([]byte, error) {
	return infoJSON(stats, stats.StatsBasic)
}

func (stats Stats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		stats.Ipackets, stats.Ibytes, stats.Imissed, stats.Ierrors, stats.Rx_nombuf, stats.Opackets, stats.Obytes, stats.Oerrors)
}
