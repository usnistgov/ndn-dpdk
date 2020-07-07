package eal

import (
	"fmt"
)

var lastObjectID uint64

// AllocObjectID allocates an identifier/name for DPDK objects.
func AllocObjectID(dbgtype string) string {
	lastObjectID++
	id := fmt.Sprintf("K%016x", lastObjectID)
	log.WithFields(makeLogFields("type", dbgtype, "id", id)).Debug("object ID allocated")
	return id
}
