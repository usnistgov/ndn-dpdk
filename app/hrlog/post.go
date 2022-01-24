// Package hrlog writes high resolution tracing logs.
package hrlog

/*
#include "../../csrc/hrlog/entry.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
)

var logger = logging.New("hrlog")

// Post posts entries to the hrlog collector.
func Post(rs *urcu.ReadSide, entries []uint64) {
	rs.Lock()
	defer rs.Unlock()

	ptr, count := cptr.ParseCptrArray(entries)
	C.Hrlog_Post((*C.HrlogEntry)(ptr), C.uint16_t(count))
}
