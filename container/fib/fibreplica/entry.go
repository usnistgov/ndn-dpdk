package fibreplica

/*
#include "../../../csrc/fib/entry.h"
#include "../../../csrc/strategyapi/api.h"

static_assert(offsetof(FibEntry, strategy) == offsetof(FibEntry, realEntry), "");
enum { c_offsetof_FibEntry_StrategyReal = offsetof(FibEntry, strategy) };

extern bool go_SgGetJSON(SgCtx* ctx, char* path, int index, int64_t* dst);
*/
import "C"
import (
	"runtime/cgo"
	"strconv"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Entry represents a FIB entry.
type Entry C.FibEntry

func entryFromPtr(c *C.FibEntry) *Entry {
	return (*Entry)(c)
}

// Ptr returns *C.FibEntry pointer.
func (entry *Entry) Ptr() unsafe.Pointer {
	return unsafe.Pointer(entry)
}

func (entry *Entry) ptr() *C.FibEntry {
	return (*C.FibEntry)(entry)
}

func (entry *Entry) ptrStrategy() **C.StrategyCode {
	return (**C.StrategyCode)(unsafe.Add(entry.Ptr(), C.c_offsetof_FibEntry_StrategyReal))
}

func (entry *Entry) ptrReal() **C.FibEntry {
	return (**C.FibEntry)(unsafe.Add(entry.Ptr(), C.c_offsetof_FibEntry_StrategyReal))
}

func (entry *Entry) ptrDyn(dynIndex int) *C.FibEntryDyn {
	return C.FibEntry_PtrDyn(entry.ptr(), C.int(dynIndex))
}

// Read converts Entry to fibdef.Entry.
func (entry *Entry) Read() (de fibdef.Entry) {
	if entry.height > 0 {
		panic("cannot Read virtual entry")
	}

	de.Name.UnmarshalBinary(cptr.AsByteSlice(entry.nameV[:entry.nameL]))

	de.Nexthops = make([]iface.ID, int(entry.nNexthops))
	for i := range de.Nexthops {
		de.Nexthops[i] = iface.ID(entry.nexthops[i])
	}

	de.Strategy = int((*entry.ptrStrategy()).id)
	return
}

// AccCounters adds to counters.
func (entry *Entry) AccCounters(cnt *fibdef.EntryCounters, t *Table) {
	for i := 0; i < t.nDyns; i++ {
		dyn := entry.Real().ptrDyn(i)
		cnt.NRxInterests += uint64(dyn.nRxInterests)
		cnt.NRxData += uint64(dyn.nRxData)
		cnt.NRxNacks += uint64(dyn.nRxNacks)
		cnt.NTxInterests += uint64(dyn.nTxInterests)
	}
}

// NexthopRtt reads RTT measurement of a nexthop.
// Return values are in TSC duration units.
func (entry *Entry) NexthopRtt(dynIndex, nexthopIndex int) (sRtt, rttVar int64) {
	dyn := entry.Real().ptrDyn(dynIndex)
	rtt := dyn.rtt[nexthopIndex]
	return int64(rtt.sRtt), int64(rtt.rttVar)
}

// IsVirt determines whether this is a virtual entry.
func (entry *Entry) IsVirt() bool {
	return entry.height > 0
}

// Real returns the real entry linked from this entry.
func (entry *Entry) Real() *Entry {
	if entry != nil && entry.IsVirt() {
		return entryFromPtr(C.FibEntry_GetReal(entry.ptr()))
	}
	return entry
}

// FibSeqNum returns the FIB insertion sequence number recorded in this entry.
func (entry *Entry) FibSeqNum() uint32 {
	return uint32(entry.seqNum)
}

func (entry *Entry) assignReal(u *fibdef.RealUpdate, sgGlobals []unsafe.Pointer) {
	entry.height = 0

	nameV, _ := u.Name.MarshalBinary()
	entry.nameL = C.uint16_t(copy(cptr.AsByteSlice(&entry.nameV), nameV))
	entry.nComps = C.uint8_t(len(u.Name))

	entry.nNexthops = C.uint8_t(len(u.Nexthops))
	for i, nh := range u.Nexthops {
		entry.nexthops[i] = C.FaceID(nh)
	}

	sc := strategycode.Get(u.Strategy)
	*entry.ptrStrategy() = (*C.StrategyCode)(sc.Ptr())

	if sgInit := sc.InitFunc(); sgInit != nil {
		var params any
		jsonhelper.Roundtrip(u.Params, &params)
		paramsHdl := cgo.NewHandle(params)
		defer paramsHdl.Delete()
		now := C.TscTime(eal.TscNow())
		for i, sgGlobal := range sgGlobals {
			ctx := (*C.FibSgInitCtx)(eal.Zmalloc("FibSgInitCtx", C.sizeof_FibSgInitCtx, eal.NumaSocket{}))
			*ctx = C.FibSgInitCtx{
				global:   (*C.SgGlobal)(sgGlobal),
				now:      now,
				entry:    entry.ptr(),
				dyn:      entry.ptrDyn(i),
				goHandle: C.uintptr_t(paramsHdl),
			}
			sgInit(unsafe.Pointer(ctx), C.sizeof_SgCtx)
		}
	}
}

func (entry *Entry) assignVirt(u *fibdef.VirtUpdate, real *Entry) {
	entry.height = C.uint8_t(u.Height)

	nameV, _ := u.Name.MarshalBinary()
	entry.nameL = C.uint16_t(copy(cptr.AsByteSlice(&entry.nameV), nameV))
	entry.nComps = C.uint8_t(len(u.Name))

	*entry.ptrReal() = real.ptr()
}

func jsonExtract(obj any, path []string, index int) (value int64, ok bool) {
	switch field := obj.(type) {
	case nil:
		return 0, false
	case map[string]any:
		if len(path) > 0 {
			return jsonExtract(field[path[0]], path[1:], index)
		}
	case []any:
		switch {
		case len(path) > 0:
			return 0, false
		case index == C.SGJSON_LEN:
			return int64(len(field)), true
		case index >= 0 && index < len(field):
			return jsonExtract(field[index], nil, C.SGJSON_SCALAR)
		}
	case bool:
		return map[bool]int64{false: 0, true: 1}[field], index == C.SGJSON_SCALAR
	case float64:
		return int64(field), index == C.SGJSON_SCALAR
	case string:
		value, e := strconv.ParseInt(field, 10, 64)
		return value, e == nil && index == C.SGJSON_SCALAR
	}
	return 0, false
}

//export go_SgGetJSON
func go_SgGetJSON(ctx0 *C.SgCtx, pathC *C.char, index C.int, dst *C.int64_t) C.bool {
	ctx := (*C.FibSgInitCtx)(unsafe.Pointer(ctx0))
	params := cgo.Handle(ctx.goHandle).Value()
	path := strings.Split(C.GoString(pathC), ".")

	value, ok := jsonExtract(params, path, int(index))
	if ok {
		*dst = C.int64_t(value)
		return true
	}
	return false
}

func init() {
	C.StrategyCode_GetJSON = C.StrategyCode_GetJSONFunc(C.go_SgGetJSON)
}
