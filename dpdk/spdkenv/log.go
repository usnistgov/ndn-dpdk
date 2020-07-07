package spdkenv

/*
#include "../../csrc/core/common.h"
#include <spdk/log.h>
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/logger"
)

var (
	log           = logger.New("spdkenv")
	makeLogFields = logger.MakeFields
)

// Set C log level from LOG_SPDK environment variable.
func initLogging() {
	lvl := logger.GetLevel("SPDK")
	lvlC := C.enum_spdk_log_level(C.SPDK_LOG_INFO)
	switch lvl {
	case 'V':
		lvlC = C.SPDK_LOG_DEBUG
	case 'D':
		lvlC = C.SPDK_LOG_INFO
	case 'I':
		lvlC = C.SPDK_LOG_NOTICE
	case 'W':
		lvlC = C.SPDK_LOG_WARN
	case 'E', 'F':
		lvlC = C.SPDK_LOG_ERROR
	case 'N':
		lvlC = C.SPDK_LOG_DISABLED
	}
	C.spdk_log_set_print_level(lvlC)
}
