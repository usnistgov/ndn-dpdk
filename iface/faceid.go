package iface

/*
#include "../csrc/iface/faceid.h"
*/
import "C"
import (
	"math/rand"
	"strconv"
)

type FaceKind int

const (
	FaceKind_None   FaceKind = -1
	FaceKind_Eth    FaceKind = 0x1
	FaceKind_Socket FaceKind = 0xE
)

var faceKinds = map[FaceKind]string{
	FaceKind_None:   "none",
	FaceKind_Eth:    "eth",
	FaceKind_Socket: "socket",
}

func (kind FaceKind) String() string {
	if s, ok := faceKinds[kind]; ok {
		return s
	}
	return strconv.Itoa(int(kind))
}

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
// Warning: endless loop if all FaceIds are used up.
func AllocId(kind FaceKind) (id FaceId) {
	for id.GetKind() != kind || gFaces[id] != nil {
		id = FaceId(kind<<12) | FaceId(rand.Uint32()&0x0FFF)
	}
	return id
}

// Allocate random FaceIds for a kind of face.
// Warning: endless loop if all FaceIds are used up.
func AllocIds(kind FaceKind, count int) (ids []FaceId) {
	allocated := make(map[FaceId]bool)
	for len(allocated) < count {
		allocated[AllocId(kind)] = true
	}
	for id := range allocated {
		ids = append(ids, id)
	}
	return ids
}

type State uint8

const (
	State_Unused  State = C.FACESTA_UNUSED
	State_Up      State = C.FACESTA_UP
	State_Down    State = C.FACESTA_DOWN
	State_Removed State = C.FACESTA_REMOVED
)

var states = map[State]string{
	State_Unused:  "unused",
	State_Up:      "up",
	State_Down:    "down",
	State_Removed: "removed",
}

func (st State) String() string {
	if s, ok := states[st]; ok {
		return s
	}
	return strconv.Itoa(int(st))
}
