package eal

import (
	"encoding/base32"
	"encoding/binary"

	"go.uber.org/zap"
)

var (
	lastObjectID uint64
	encObjectID  = base32.HexEncoding.WithPadding(base32.NoPadding)
)

// AllocObjectID allocates an identifier/name for DPDK objects.
func AllocObjectID(dbgtype string) string {
	lastObjectID++

	var bin [8]uint8
	binary.BigEndian.PutUint64(bin[:], lastObjectID)

	var b32 [14]uint8
	b32[0] = 'W'
	encObjectID.Encode(b32[1:], bin[:])

	logger.Debug("object ID allocated",
		zap.String("type", dbgtype),
		zap.ByteString("id", b32[:]),
	)
	return string(b32[:])
}
