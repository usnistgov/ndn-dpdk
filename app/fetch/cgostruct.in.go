// +build ignore

package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"

// SegState contains per-segment state.
type SegState C.FetchSeg

type fetchRetxNode C.FetchRetxNode
type fetchRetxQueue C.FetchRetxQueue
type minSched C.MinSched
type minTmr C.MinTmr
