package ethport

/*
#include "../../csrc/ethface/gtpip-table.h"

static int c_rte_hash_add_key_data(const struct rte_hash* h, const void* key, uint64_t data)
{
	return rte_hash_add_key_data(h, key, (void*)data);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"net/netip"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/zyedidia/generic"
)

// GtpipTableConfig contains GTP-IP table configuration.
type GtpipTableConfig struct {
	IPv4Capacity int `json:"ipv4capacity,omitempty"`
}

// GtpipTable represents GTP-IP table.
type GtpipTable C.GtpipTable

// Insert inserts a record of UE IP Address and GTP-U face.
func (table *GtpipTable) Insert(ueIP netip.Addr, face iface.Face) error {
	switch {
	case ueIP.Unmap().Is4():
		key := ueIP.As4()
		hdata := uintptr(face.ID())
		res := C.c_rte_hash_add_key_data(table.ipv4, unsafe.Pointer(&key), C.uint64_t(hdata))
		return eal.MakeErrno(res)
	}
	return errors.New("not IPv4 address")
}

// Delete deletes a record of UE IP Address.
func (table *GtpipTable) Delete(ueIP netip.Addr) error {
	switch {
	case ueIP.Unmap().Is4():
		key := ueIP.As4()
		res := C.rte_hash_del_key(table.ipv4, unsafe.Pointer(&key))
		if res >= 0 {
			return nil
		}
		return eal.MakeErrno(res)
	}
	return errors.New("not IPv4 address")
}

func (table *GtpipTable) ProcessDownlink(pkt *pktmbuf.Packet) bool {
	return bool(C.GtpipTable_ProcessDownlink((*C.GtpipTable)(table), (*C.struct_rte_mbuf)(pkt.Ptr())))
}

func (table *GtpipTable) ProcessUplink(pkt *pktmbuf.Packet) bool {
	return bool(C.GtpipTable_ProcessUplink((*C.GtpipTable)(table), (*C.struct_rte_mbuf)(pkt.Ptr())))
}

// Close deletes the table.
func (table *GtpipTable) Close() error {
	if table.ipv4 != nil {
		C.rte_hash_free(table.ipv4)
	}
	eal.Free(table)
	return nil
}

// NewGtpipTable creates a GTP-IP table.
func NewGtpipTable(cfg GtpipTableConfig, socket eal.NumaSocket) (table *GtpipTable, e error) {
	table = (*GtpipTable)(eal.Zmalloc[C.GtpipTable]("GtpipTable", C.sizeof_GtpipTable, socket))

	ht4ID := C.CString(eal.AllocObjectID("gtpip.GtpipTable.ipv4"))
	defer C.free(unsafe.Pointer(ht4ID))
	cap4 := generic.Clamp(cfg.IPv4Capacity, 256, 65536)
	if table.ipv4 = C.HashTable_New(C.struct_rte_hash_parameters{
		name:       ht4ID,
		entries:    C.uint32_t(cap4),
		key_len:    C.sizeof_uint32_t,
		socket_id:  C.int(socket.ID()),
		extra_flag: C.RTE_HASH_EXTRA_FLAGS_RW_CONCURRENCY | C.RTE_HASH_EXTRA_FLAGS_EXT_TABLE,
	}); table.ipv4 == nil {
		e := eal.GetErrno()
		table.Close()
		return nil, fmt.Errorf("HashTable_New failed: %w", e)
	}

	return table, nil
}
