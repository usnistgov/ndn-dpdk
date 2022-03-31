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

	C.Hrlog_Post(cptr.FirstPtr[C.HrlogEntry](entries), C.uint16_t(len(entries)))
}
