#ifndef NDNDPDK_FWDP_DISK_H
#define NDNDPDK_FWDP_DISK_H

/** @file */

#include "../iface/input-demux.h"

/** @brief Forwarder data plane, disk helper. */
typedef struct FwDisk {
  InputDemux output;
} FwDisk;

/**
 * @brief Handle DiskStore_GetData completion.
 * @param npkt Interest packet.
 * @param ctx FwDisk* pointer.
 */
__attribute__((nonnull)) void
FwDisk_GotData(Packet* npkt, uintptr_t ctx);

#endif // NDNDPDK_FWDP_DISK_H
