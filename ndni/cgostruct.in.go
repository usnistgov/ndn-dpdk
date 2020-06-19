// +build ignore

package ndni

/*
#include "../csrc/ndn/packet.h"
*/
import "C"

// LpL3 contains NDNLPv2 layer 3 header fields.
type LpL3 C.LpL3

// LpL2 contains NDNLPv2 layer 2 header fields.
type LpL2 C.LpL2

// LpHeader is LpL2 + LpL3.
type LpHeader C.LpHeader

// LName is a name in linear mbuf.
type LName C.LName

// PName is a parsed name.
type PName C.PName

// CName is PName + buffer pointer.
type CName C.Name

// InterestTemplate is a template to encode Interests.
type InterestTemplate C.InterestTemplate

type pInterest C.PInterest
type pData C.PData
type pNack C.PNack
