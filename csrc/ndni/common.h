#ifndef NDNDPDK_NDNI_COMMON_H
#define NDNDPDK_NDNI_COMMON_H

/** @file */

#include "../core/common.h"
#include <rte_byteorder.h>

#include "../dpdk/cryptodev.h"
#include "../dpdk/mbuf.h"

#include "an.h"
#include "enum.h"

typedef struct Packet Packet;
typedef struct PInterest PInterest;
typedef struct PData PData;
typedef struct PNack PNack;

/** @brief Mempools for packet modification. */
typedef struct PacketMempools
{
  struct rte_mempool* packet;
  struct rte_mempool* indirect;
  struct rte_mempool* header;
} PacketMempools;

#endif // NDNDPDK_NDNI_COMMON_H
