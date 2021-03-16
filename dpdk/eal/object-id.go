package eal

import (
	"fmt"

	"go.uber.org/zap"
)

var lastObjectID uint64

// AllocObjectID allocates an identifier/name for DPDK objects.
func AllocObjectID(dbgtype string) string {
	lastObjectID++
	id := fmt.Sprintf("K%016x", lastObjectID)
	logger.Debug("object ID allocated",
		zap.String("type", dbgtype),
		zap.String("id", id),
	)
	return id
}
