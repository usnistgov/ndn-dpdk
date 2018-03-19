package ifacetest

import (
	"sync"
	"testing"

	"ndn-dpdk/iface"
)

func TestFaceTable(t *testing.T) {
	assert, _ := makeAR(t)

	ft := iface.NewFaceTable()
	assert.Equal(0, ft.Len())
	assert.Len(ft.ListFaces(), 0)
	clearNFaceTableTestFaceCloses()

	const NFACES = 1000
	var faces [NFACES + 1]FaceTableTestFace

	var wg sync.WaitGroup
	wg.Add(NFACES)
	for i := 1; i <= NFACES; i++ {
		go func(id iface.FaceId) {
			assert.False(ft.GetFace(id).IsValid())
			face := newFaceTableTestFace(id)
			ft.AddFace(face.Face)
			assert.Equal(face.GetPtr(), ft.GetFace(id).GetPtr())
			faces[id] = face
			wg.Done()
		}(iface.FaceId(i))
	}

	wg.Wait()
	assert.Equal(NFACES, ft.Len())
	assert.Len(ft.ListFaces(), 1000)
	assert.Equal(0, getNFaceTableTestFaceCloses())

	wg.Add(NFACES)
	for i := 1; i <= NFACES; i++ {
		go func(id iface.FaceId) {
			assert.Equal(faces[id].GetPtr(), ft.GetFace(id).GetPtr())
			ft.RemoveFace(id)
			assert.False(ft.GetFace(id).IsValid())
			wg.Done()
		}(iface.FaceId(i))
	}

	wg.Wait()
	assert.Equal(0, ft.Len())
	assert.Equal(NFACES, getNFaceTableTestFaceCloses())
}
