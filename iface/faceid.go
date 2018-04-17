package iface

/*
#include "faceid.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
)

type FaceKind int

const (
	FaceKind_None   FaceKind = -1
	FaceKind_Mock   FaceKind = 0x0
	FaceKind_Eth    FaceKind = 0x1
	FaceKind_Socket FaceKind = 0xE
)

var faceKinds = map[FaceKind]string{
	FaceKind_None:   "none",
	FaceKind_Mock:   "mock",
	FaceKind_Eth:    "eth",
	FaceKind_Socket: "socket",
}

func (kind FaceKind) String() string {
	if s, ok := faceKinds[kind]; ok {
		return s
	}
	return fmt.Sprintf("%d", kind)
}

// Numeric face identifier, may appear in rte_mbuf.port field
type FaceId uint16

const (
	FACEID_INVALID FaceId = C.FACEID_INVALID
	FACEID_MIN     FaceId = C.FACEID_MIN
	FACEID_MAX     FaceId = C.FACEID_MAX
)

func (id FaceId) GetKind() FaceKind {
	if id == FACEID_INVALID {
		return FaceKind_None
	}
	kind := FaceKind(id >> 12)
	if _, ok := faceKinds[kind]; ok {
		return kind
	}
	return FaceKind_None
}

// Allocate a random FaceId for a kind of face.
func AllocId(kind FaceKind) (id FaceId) {
	for id.GetKind() != kind || gFaces[id] != nil {
		id = FaceId(kind<<12) | FaceId(rand.Uint32()&0x0FFF)
	}
	return id
}
