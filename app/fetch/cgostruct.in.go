//go:build ignore

package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"

// SegState contains per-segment state.
type SegState C.FetchSeg

type cdsListHead C.struct_cds_list_head
type minSched C.MinSched
type minTmr C.MinTmr
