#ifndef NDNDPDK_NDNI_NACK_H
#define NDNDPDK_NDNI_NACK_H

/** @file */

#include "interest.h"
#include "lp.h"

/** @brief Return the less severe NackReason. */
static inline NackReason
NackReason_GetMin(NackReason a, NackReason b)
{
  return RTE_MIN(a, b);
}

__attribute__((returns_nonnull)) const char*
NackReason_ToString(NackReason reason);

/** @brief Parsed Nack packet. */
typedef struct PNack
{
  LpL3 lpl3;
  PInterest interest;
} PNack;

/**
 * @brief Turn an Interest into a Nack.
 * @param npkt a packet of type @c PktInterest or @c PktSInterest .
 *             Its first segment must be a uniquely owned direct mbuf.
 * @retval NULL allocation failure.
 * @return Nack packet. It may be different from @p npkt .
 * @pre PktType is @c PktInterest or @c PktSInterest .
 * @post PktType is @c PktSNack .
 */
__attribute__((nonnull)) Packet*
Nack_FromInterest(Packet* npkt, NackReason reason, PacketMempools* mp, PacketTxAlign align);

#endif // NDNDPDK_NDNI_NACK_H
