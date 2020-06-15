#ifndef NDN_DPDK_NDN_NACK_H
#define NDN_DPDK_NDN_NACK_H

/// \file

#include "interest.h"
#include "lp.h"

/** \brief Indicate a Nack reason.
 */
typedef enum NackReason
{
  NackReason_None = 0, ///< packet is not a Nack
  NackReason_Congestion = 50,
  NackReason_Duplicate = 100,
  NackReason_NoRoute = 150,
  NackReason_Unspecified = 255, ///< reason unspecified
  NackReason_Max = 255,
} NackReason;

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

static inline NackReason
PNack_GetReason(const PNack* nack)
{
  return nack->lpl3.nackReason;
}

/** \brief Turn an Interest into a Nack.
 *  \pre Packet_GetL3PktType(npkt) == L3PktType_Interest
 *  \post Packet_GetL3PktType(npkt) == L3PktType_Nack
 */
void
MakeNack(Packet* npkt, NackReason reason);

#endif // NDN_DPDK_NDN_NACK_H
