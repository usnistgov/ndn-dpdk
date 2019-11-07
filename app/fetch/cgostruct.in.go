// +build ignore

package fetch

/*
#include "rttest.h"
#include "tcpcubic.h"
#include "window.h"
*/
import "C"

type RttEst C.RttEst

// Per-segment state.
type FetchSeg C.FetchSeg

// Window of segment states.
type FetchWindow C.FetchWindow

// TCP CUBIC algorithm.
type TcpCubic C.TcpCubic
