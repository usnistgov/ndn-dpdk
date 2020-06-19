#ifndef NDN_DPDK_NDN_NACK_H
#define NDN_DPDK_NDN_NACK_H

/// \file

#include "interest.h"
#include "lp.h"

/** \brief Return the less severe NackReason.
 */
static inline NackReason
NackReason_GetMin(NackReason a, NackReason b)
{
  return RTE_MIN(a, b);
}

const char*
NackReason_ToString(NackReason reason);

/** \brief Parsed Nack packet.
 */
typedef struct PNack
{
  LpL3 lpl3;
  PInterest interest;
} PNack;

/** \brief Turn an Interest into a Nack.
 *  \pre Packet_GetL3PktType(npkt) == L3PktTypeInterest
 *  \post Packet_GetL3PktType(npkt) == L3PktTypeNack
 */
void
MakeNack(Packet* npkt, NackReason reason);

#endif // NDN_DPDK_NDN_NACK_H
