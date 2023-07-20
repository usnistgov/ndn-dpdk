#ifndef NDNDPDK_NDNI_COMMON_H
#define NDNDPDK_NDNI_COMMON_H

/** @file */

#include "../dpdk/cryptodev.h"
#include "../dpdk/mbuf.h"
#include "an.h"
#include "enum.h"

typedef struct Packet Packet;
typedef struct PInterest PInterest;
typedef struct PData PData;
typedef struct PNack PNack;

/** @brief Mempools for packet modification. */
typedef struct PacketMempools {
  struct rte_mempool* packet;
  struct rte_mempool* indirect;
  struct rte_mempool* header;
} PacketMempools;

/**
 * @brief mbuf alignment requirements for encoding or packet modification.
 *
 * If @c linearize is set to true, encoders should output direct mbufs, copying payload when
 * necessary. Each @c mbuf should have at least @c RTE_PKTMBUF_HEADROOM+LpHeaderHeadroom headroom
 * and its @c data_len cannot exceed @c fragmentPayloadSize . FaceTx will prepend NDNLPv2 headers
 * to each mbuf and transmit it as a NDNLPv2 fragment.
 *
 * If @c linearize is set to false, encoders may use a mix of direct and indirect mbufs with
 * arbitrary boundaries. FaceTx will perform fragmentation as needed. @c fragmentPayloadSize
 * is ignored.
 */
typedef struct PacketTxAlign {
  /** @brief Max payload size per fragment. */
  uint16_t fragmentPayloadSize;

  /** @brief Whether packet must be linearized into contiguous mbufs. */
  bool linearize;
} PacketTxAlign;

#endif // NDNDPDK_NDNI_COMMON_H
