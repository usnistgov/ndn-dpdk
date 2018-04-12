package iface

/*
#include "faceid.h"
*/
import "C"

type FaceKind int

const (
	FaceKind_None FaceKind = iota
	FaceKind_Mock
	FaceKind_Eth
	FaceKind_Socket
)

// Numeric face identifier, may appear in rte_mbuf.port field
type FaceId uint16

const (
	FACEID_INVALID FaceId = C.FACEID_INVALID
	FACEID_MIN     FaceId = C.FACEID_MIN
	FACEID_MAX     FaceId = C.FACEID_MAX
)

func (id FaceId) GetKind() FaceKind {
	switch id >> 12 {
	case 0x0:
		if id == FACEID_INVALID {
			return FaceKind_None
		}
		return FaceKind_Mock
	case 0x1:
		return FaceKind_Eth
	case 0xE:
		return FaceKind_Socket
	}
	return FaceKind_None
}
