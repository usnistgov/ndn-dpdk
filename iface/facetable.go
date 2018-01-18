package iface

/*
#include "facetable.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type FaceTable struct {
	c *C.FaceTable
}

func NewFaceTable() (ft FaceTable) {
	ft.c = (*C.FaceTable)(dpdk.Zmalloc("FaceTable", C.sizeof_FaceTable,
		dpdk.GetMasterLCore().GetNumaSocket()))
	return ft
}

// Get native *C.FaceTable pointer to use in other packages.
func (ft FaceTable) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(ft.c)
}

func (ft FaceTable) Len() int {
	return int(C.FaceTable_Count(ft.c))
}

func (ft FaceTable) ListFaces() []Face {
	faces := make([]Face, 0)
	for i := FACEID_MIN; i < FACEID_MAX; i++ {
		face := ft.GetFace(i)
		if face.IsValid() {
			faces = append(faces, face)
		}
	}
	return faces
}

func (ft FaceTable) GetFace(id FaceId) Face {
	return FaceFromPtr(unsafe.Pointer(C.FaceTable_GetFace(ft.c, C.FaceId(id))))
}

func (ft FaceTable) SetFace(face Face) {
	C.FaceTable_SetFace(ft.c, (*C.Face)(face.GetPtr()))
}

func (ft FaceTable) UnsetFace(id FaceId) {
	C.FaceTable_UnsetFace(ft.c, C.FaceId(id))
}
