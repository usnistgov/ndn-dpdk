package ethport

/*
#include "../../csrc/ethface/gtpip.h"

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

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/zyedidia/generic"
)

// GtpipConfig contains GTP-IP handler configuration.
type GtpipConfig struct {
	// UE IPv4 address hashtable capacity, between 256 and 65536.
	IPv4Capacity int `json:"ipv4capacity,omitempty"`
}

// Gtpip represents GTP-IP handler.
type Gtpip C.EthGtpip

// Len returns number of entries.
func (g *Gtpip) Len() int {
	var n C.int32_t
	n += C.rte_hash_count(g.ipv4)
	return int(n)
}

// Insert inserts a record of UE IP Address and GTP-U face.
func (g *Gtpip) Insert(ueIP netip.Addr, face iface.Face) error {
	switch {
	case ueIP.Unmap().Is4():
		key := ueIP.As4()
		hdata := uintptr(face.ID())
		res := C.c_rte_hash_add_key_data(g.ipv4, unsafe.Pointer(&key), C.uint64_t(hdata))
		return eal.MakeErrno(res)
	}
	return errors.New("not IPv4 address")
}

// Delete deletes a record of UE IP Address.
func (g *Gtpip) Delete(ueIP netip.Addr) error {
	switch {
	case ueIP.Unmap().Is4():
		key := ueIP.As4()
		res := C.rte_hash_del_key(g.ipv4, unsafe.Pointer(&key))
		if res >= 0 {
			return nil
		}
		return eal.MakeErrno(res)
	}
	return errors.New("not IPv4 address")
}

// ProcessDownlink processes a burst of downlink Ethernet frames.
func (g *Gtpip) ProcessDownlink(vec pktmbuf.Vector) []bool {
	return cptr.MapInChunksOf(64, vec, func(vec pktmbuf.Vector) []bool {
		mask := C.EthGtpip_ProcessDownlinkBulk((*C.EthGtpip)(g), cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint32_t(len(vec)))
		return cptr.ExpandBits(len(vec), mask)
	})
}

// ProcessUplink processes a burst of uplink Ethernet frames.
func (g *Gtpip) ProcessUplink(vec pktmbuf.Vector) []bool {
	return cptr.MapInChunksOf(64, vec, func(vec pktmbuf.Vector) []bool {
		mask := C.EthGtpip_ProcessUplinkBulk((*C.EthGtpip)(g), cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint32_t(len(vec)))
		return cptr.ExpandBits(len(vec), mask)
	})
}

// Close releases memory.
func (g *Gtpip) Close() error {
	if g.ipv4 != nil {
		C.rte_hash_free(g.ipv4)
	}
	eal.Free(g)
	return nil
}

// NewGtpip creates a GTP-IP handler.
func NewGtpip(cfg GtpipConfig, socket eal.NumaSocket) (g *Gtpip, e error) {
	g = (*Gtpip)(eal.Zmalloc[C.EthGtpip]("EthGtpip", C.sizeof_EthGtpip, socket))

	ht4ID := C.CString(eal.AllocObjectID("ethport.Gtpip.ipv4"))
	defer C.free(unsafe.Pointer(ht4ID))
	if g.ipv4 = C.HashTable_New(C.struct_rte_hash_parameters{
		name:       ht4ID,
		entries:    C.uint32_t(generic.Clamp(cfg.IPv4Capacity, 256, 65536)),
		key_len:    C.sizeof_uint32_t,
		socket_id:  C.int(socket.ID()),
		extra_flag: C.RTE_HASH_EXTRA_FLAGS_RW_CONCURRENCY | C.RTE_HASH_EXTRA_FLAGS_EXT_TABLE,
	}); g.ipv4 == nil {
		e := eal.GetErrno()
		g.Close()
		return nil, fmt.Errorf("HashTable_New failed: %w", e)
	}

	return g, nil
}
