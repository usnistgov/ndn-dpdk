package ethnetif

import (
	"math"
	"reflect"

	"github.com/dylandreimerink/gobpfld"
	"github.com/dylandreimerink/gobpfld/bpfsys"
	"github.com/dylandreimerink/gobpfld/bpftypes"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"go.uber.org/zap"
)

var xdpDevs = map[int]*xdpDev{}

type xdpDev struct {
	n       *NetIntf
	faceMap *gobpfld.HashMap
}

// Close closes face_map if it was opened.
func (xd *xdpDev) Close() error {
	if xd.faceMap != nil {
		xd.faceMap.Close()
		xd.faceMap = nil
	}
	return nil
}

// FindFaceMap opens face_map defined in the XDP program.
// Returns nil if xd is nil.
func (xd *xdpDev) FindFaceMap() *gobpfld.HashMap {
	if xd == nil {
		return nil
	}
	if xd.faceMap != nil {
		return xd.faceMap
	}

	logEntry := logger.With(zap.String("netif", xd.n.Name))
	xd.n.Refresh()
	if xd.n.Xdp == nil || xd.n.Xdp.ProgId == math.MaxUint32 {
		logEntry.Warn("netif has no attached XDP program")
		return nil
	}

	programs, e := gobpfld.GetLoadedPrograms()
	if e != nil {
		logEntry.Warn("gobpfld.GetLoadedPrograms error", zap.Error(e))
		return nil
	}

	var p *gobpfld.BPFProgInfo
	for _, program := range programs {
		if program.Type == bpftypes.BPF_PROG_TYPE_XDP && program.ID == xd.n.Xdp.ProgId {
			p = &program
			break
		}
	}
	if p == nil {
		logEntry.Warn("gobpfld.BPFProgInfo not found", zap.Uint32("prog-id", xd.n.Xdp.ProgId))
		return nil
	}

	for _, mid := range p.MapIDs {
		if hash := xdpMatchHashMap(mid, "face_map"); hash != nil {
			xd.faceMap = hash
			break
		}
	}
	if xd.faceMap == nil {
		logEntry.Warn("BPF program does not define face_map")
	}

	return xd.faceMap
}

func xdpMatchHashMap(mid uint32, wantName string) (hash *gobpfld.HashMap) {
	logEntry := logger.With(zap.Uint32("map-id", mid))

	m, e := gobpfld.MapFromID(mid)
	if e != nil {
		logEntry.Warn("gobpfld.MapFromID error", zap.Error(e))
		return nil
	}

	name, def := m.GetName(), m.GetDefinition()
	if def.Type != bpftypes.BPF_MAP_TYPE_HASH || name.String() != wantName {
		m.Close()
		return nil
	}

	// don't call m.Load() - it would make a copy of the map instead of operating on the existing map
	return m.(*gobpfld.HashMap)
}

// XDPInsertFaceMapEntry inserts an entry in the FaceMap defined in the XDP program attached to the EthDev.
// If the EthDev is not using XDP driver, this operation has no effect.
func XDPInsertFaceMapEntry(dev ethdev.EthDev, key []byte, xskQueue int) {
	fm := xdpDevs[dev.ID()].FindFaceMap()
	if fm == nil {
		return
	}

	logEntry := logger.With(
		zap.Uint32("map-fd", uint32(fm.GetFD())),
		zap.Binary("key", key),
		zap.Int("xsk-queue", xskQueue),
	)

	keyPtr := xdpHashMakeKey(fm.GetDefinition().KeySize, key)
	value := int32(xskQueue)
	if e := fm.Set(keyPtr, &value, bpfsys.BPFMapElemAny); e != nil {
		logEntry.Warn("HashMap.Set error", zap.Error(e))
	} else {
		logEntry.Debug("HashMap.Set success")
	}
}

// XDPDeleteFaceMapEntry deletes an entry in the FaceMap defined in the XDP program attached to the EthDev.
// If the EthDev is not using XDP driver, this operation has no effect.
func XDPDeleteFaceMapEntry(dev ethdev.EthDev, key []byte) {
	fm := xdpDevs[dev.ID()].FindFaceMap()
	if fm == nil {
		return
	}

	logEntry := logger.With(
		zap.Uint32("map-fd", uint32(fm.GetFD())),
		zap.Binary("key", key),
	)

	keyPtr := xdpHashMakeKey(fm.GetDefinition().KeySize, key)
	if e := fm.Delete(keyPtr); e != nil {
		logEntry.Warn("HashMap.Delete error", zap.Error(e))
	} else {
		logEntry.Debug("HashMap.Delete success")
	}
}

// xdpHashMakeKey returns *[size]byte pointer as needed by gobpfld.AbstractMap.toKeyPtr().
func xdpHashMakeKey(size uint32, key []byte) (arrayPtr any) {
	ptr := reflect.New(reflect.ArrayOf(int(size), reflect.TypeFor[byte]()))
	copy(ptr.Elem().Slice(0, int(size)).Interface().([]byte), key)
	return ptr.Interface()
}
