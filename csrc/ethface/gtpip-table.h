#ifndef NDNDPDK_ETHFACE_GTPIP_TABLE_H
#define NDNDPDK_ETHFACE_GTPIP_TABLE_H

/** @file */

#include "../dpdk/hashtable.h"
#include "../iface/face.h"

/** @brief GTP-IP table. */
typedef struct GtpipTable {
  struct rte_hash* ipv4;
} GtpipTable;

__attribute__((nonnull)) bool
GtpipTable_ProcessDownlink(GtpipTable* table, struct rte_mbuf* m);

__attribute__((nonnull)) bool
GtpipTable_ProcessUplink(GtpipTable* table, struct rte_mbuf* m);

#endif // NDNDPDK_ETHFACE_GTPIP_TABLE_H
