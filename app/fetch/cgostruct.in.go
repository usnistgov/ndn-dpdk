// +build ignore

package fetch

/*
#include "window.h"
*/
import "C"

// Per-segment state.
type FetchSeg C.FetchSeg

// Window of segment states.
type FetchWindow C.FetchWindow
