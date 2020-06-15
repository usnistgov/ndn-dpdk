// +build ignore

package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"

// RTT estimator.
type RttEst C.RttEst

// Per-segment state.
type SegState C.struct_FetchSeg

// 'struct_' is necessary to workaround https://github.com/golang/go/issues/37479 for Go 1.14

// Window of segment states.
type Window C.FetchWindow

// TCP CUBIC algorithm.
type TcpCubic C.TcpCubic

// Fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

type fetchRetxNode C.FetchRetxNode
type fetchRetxQueue C.FetchRetxQueue
type minSched C.MinSched
type minTmr C.MinTmr
