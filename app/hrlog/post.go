// Package hrlog writes high resolution tracing logs.
package hrlog

/*
#include "../../csrc/hrlog/entry.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
)

var logger = logging.New("hrlog")

// Post posts entries to the hrlog collector.
func Post(entries []uint64) {
	ptr, count := cptr.ParseCptrArray(entries)
	C.Hrlog_Post((*C.HrlogEntry)(ptr), C.uint16_t(count))
}
