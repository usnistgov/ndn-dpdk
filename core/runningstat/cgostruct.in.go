//go:build ignore

package runningstat

/*
#include "../../csrc/core/running-stat.h"
*/
import "C"

type runningStat C.RunningStat
