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

// Set C log level from NDNDPDK_LOG_SPDK environment variable.
func initLogging() {
	C.spdk_log_set_print_level(func() C.enum_spdk_log_level {
		switch logger.GetLevel("SPDK") {
		case 'V':
			return C.SPDK_LOG_DEBUG
		case 'D':
			return C.SPDK_LOG_INFO
		case 'I':
			return C.SPDK_LOG_NOTICE
		case 'W':
			return C.SPDK_LOG_WARN
		case 'E', 'F':
			return C.SPDK_LOG_ERROR
		case 'N':
			return C.SPDK_LOG_DISABLED
		}
		return C.SPDK_LOG_INFO
	}())
}
