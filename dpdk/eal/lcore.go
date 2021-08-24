package eal

/*
#include "../../csrc/core/common.h"

#include <rte_launch.h>
#include <rte_lcore.h>
*/
import "C"
import (
	"encoding/json"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MaxLCoreID is the (exclusive) maximum LCore ID.
const MaxLCoreID = C.RTE_MAX_LCORE

var (
	lcoreIsBusy   [MaxLCoreID]bool
	lcoreToSocket [MaxLCoreID]int
)

// LCore represents a logical core.
// Zero LCore is invalid.
type LCore struct {
	v int // lcore ID + 1
}

// LCoreFromID converts lcore ID to LCore.
func LCoreFromID(id int) (lc LCore) {
	if id < 0 || id >= C.RTE_MAX_LCORE {
		return lc
	}
	lc.v = id + 1
	return lc
}

// CurrentLCore returns the current lcore.
func CurrentLCore() LCore {
	return LCoreFromID(int(C.rte_lcore_id()))
}

// ID returns lcore ID.
func (lc LCore) ID() int {
	return lc.v - 1
}

// Valid returns true if this is a valid lcore (not zero value).
func (lc LCore) Valid() bool {
	return lc.v != 0
}

func (lc LCore) String() string {
	if !lc.Valid() {
		return "invalid"
	}
	return strconv.Itoa(lc.ID())
}

// MarshalJSON encodes lcore as number.
// Invalid lcore is encoded as null.
func (lc LCore) MarshalJSON() ([]byte, error) {
	if !lc.Valid() {
		return json.Marshal(nil)
	}
	return json.Marshal(lc.ID())
}

// ZapField returns a zap.Field for logging.
func (lc LCore) ZapField(key string) zap.Field {
	if !lc.Valid() {
		return zap.String(key, "invalid")
	}
	return zap.Int(key, lc.ID())
}

// NumaSocket returns the NUMA socket where this lcore is located.
func (lc LCore) NumaSocket() (socket NumaSocket) {
	if !lc.Valid() {
		return socket
	}
	return NumaSocketFromID(lcoreToSocket[lc.ID()])
}

// IsBusy returns true if this lcore is running a function.
func (lc LCore) IsBusy() bool {
	if !lc.Valid() {
		return true
	}
	return lcoreIsBusy[lc.ID()]
}

// RemoteLaunch asynchronously launches a function on this lcore.
// Errors are fatal.
func (lc LCore) RemoteLaunch(fn cptr.Function) {
	if !lc.Valid() {
		logger.Panic("invalid lcore")
	}
	lcoreIsBusy[lc.ID()] = true
	PostMain(cptr.Func0.Void(func() {
		f, ctx := cptr.Func0.CallbackOnce(fn)
		res := C.rte_eal_remote_launch((*C.lcore_function_t)(f), unsafe.Pointer(ctx), C.uint(lc.ID()))
		if res != 0 {
			logger.Fatal("RemoteLaunch error", zap.Error(MakeErrno(res)))
		}
	}))
}

// Wait blocks until this lcore finishes running, and returns lcore function's return value.
// If this lcore is not running, returns 0 immediately.
func (lc LCore) Wait() (exitCode int) {
	CallMain(func() {
		exitCode = int(C.rte_eal_wait_lcore(C.uint(lc.ID())))
	})
	lcoreIsBusy[lc.ID()] = false
	return exitCode
}

// LCorePredicate is a predicate function on LCore.
type LCorePredicate func(lc LCore) bool

// Negate returns the negated predicate.
func (pred LCorePredicate) Negate() LCorePredicate {
	return func(lc LCore) bool {
		return !pred(lc)
	}
}

// LCoreOnNumaSocket creates a predicate that accepts lcores on a particular NUMA socket.
// If socket==any, it accepts any NUMA socket.
func LCoreOnNumaSocket(socket NumaSocket) LCorePredicate {
	if socket.IsAny() {
		return func(lc LCore) bool { return true }
	}
	return func(lc LCore) bool {
		return lc.NumaSocket() == socket
	}
}

// LCores is a slice of LCore.
type LCores []LCore

// MarshalLogArray implements zapcore.ArrayMarshaler interface.
func (lcores LCores) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, lc := range lcores {
		if lc.Valid() {
			enc.AppendInt(lc.ID())
		} else {
			enc.AppendString("invalid")
		}
	}
	return nil
}

// ByNumaSocket classifies lcores by NUMA socket.
func (lcores LCores) ByNumaSocket() (m map[NumaSocket]LCores) {
	m = map[NumaSocket]LCores{}
	for _, lc := range lcores {
		socket := lc.NumaSocket()
		m[socket] = append(m[socket], lc)
	}
	return m
}

// Filter returns lcores that satisfy zero or more predicates.
func (lcores LCores) Filter(pred ...LCorePredicate) (filtered LCores) {
	for _, lc := range lcores {
		ok := true
		for _, f := range pred {
			ok = ok && f(lc)
		}
		if ok {
			filtered = append(filtered, lc)
		}
	}
	return filtered
}
