#ifndef NDNDPDK_NDNI_LP_H
#define NDNDPDK_NDNI_LP_H

/** @file */

#include "common.h"

/** @brief NDNLPv2 layer 2 fields. */
typedef struct LpL2
{
  uint64_t seqNum;
  uint16_t fragIndex;
  uint16_t fragCount;
} LpL2;

/** @brief NDNLPv2 layer 3 fields. */
typedef struct LpL3
{
  uint64_t pitToken;
  uint8_t nackReason;
  uint8_t congMark;
} LpL3;

/** @brief Parsed NDNLPv2 header. */
typedef struct LpHeader
{
  LpL3 l3;
  LpL2 l2;
} LpHeader;

/**
 * @brief Parse NDNLPv2 header and strip from mbuf.
 * @param pkt a uniquely owned, unsegmented, direct mbuf.
 * @return whether success.
 * @post @p pkt contains only (fragment of) network layer packet.
 *
 * This function recognizes these NDNLPv2 features:
 * @li indexed fragmentation-reassembly
 * @li PIT token
 * @li network nack
 * @li congestion mark
 *
 * This function does not check whether header fields are applicable to network layer packet type,
 * because network layer type is unknown before reassembly. For example, it would accept Nack
 * header on Data packet.
 */
__attribute__((nonnull)) bool
LpHeader_Parse(LpHeader* lph, struct rte_mbuf* pkt);

/**
 * @brief Prepend NDNLPv2 header to mbuf.
 * @param pkt target mbuf, must have enough headroom.
 * @pre @p pkt contains (fragment of) network layer packet.
 * @post @p pkt contains LpPacket.
 */
__attribute__((nonnull)) void
LpHeader_Prepend(struct rte_mbuf* pkt, const LpL3* l3, const LpL2* l2);

#endif // NDNDPDK_NDN_LP_H
