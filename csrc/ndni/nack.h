#ifndef NDN_DPDK_NDNI_NACK_H
#define NDN_DPDK_NDNI_NACK_H

/** @file */

#include "interest.h"
#include "lp.h"

/** @brief Return the less severe NackReason. */
static inline NackReason
NackReason_GetMin(NackReason a, NackReason b)
{
  return RTE_MIN(a, b);
}

const char*
NackReason_ToString(NackReason reason);

/** @brief Parsed Nack packet. */
typedef struct PNack
{
  LpL3 lpl3;
  PInterest interest;
} PNack;

/**
 * @brief Turn an Interest into a Nack.
 * @param npkt a packet of type PktInterest or PktSInterest.
 *             Its first segment must be a uniquely owned direct mbuf.
 * @return @p npkt .
 * @pre PktType is PktInterest or PktSInterest
 * @post PktType is PktNack or PktSNack
 */
__attribute__((nonnull, returns_nonnull)) Packet*
Nack_FromInterest(Packet* npkt, NackReason reason);

#endif // NDN_DPDK_NDNI_NACK_H
