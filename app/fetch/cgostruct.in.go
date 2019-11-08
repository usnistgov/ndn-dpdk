// +build ignore

package fetch

/*
#include "logic.h"
*/
import "C"

// RTT estimator.
type RttEst C.RttEst

// Per-segment state.
type SegState C.FetchSeg

// Window of segment states.
type Window C.FetchWindow

// TCP CUBIC algorithm.
type TcpCubic C.TcpCubic

// Fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

type fetchRetxQueue C.FetchRetxQueue
type minSched C.MinSched
type minTmr C.MinTmr
