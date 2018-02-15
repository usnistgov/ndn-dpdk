#ifndef NDN_DPDK_NDN_NACK_H
#define NDN_DPDK_NDN_NACK_H

/// \file

#include "interest.h"
#include "lp.h"

/** \brief Indicate a Nack reason.
 */
typedef enum NackReason {
  NackReason_None = 0, ///< packet is not a Nack
  NackReason_Congestion = 50,
  NackReason_Duplicate = 100,
  NackReason_NoRoute = 150,
  NackReason_Unspecified = 255 ///< reason unspecified
} NackReason;

/** \brief Parsed Nack packet.
 */
typedef struct PNack
{
  LpL3 lpl3;
  PInterest interest;
} PNack;

static NackReason
PNack_GetReason(const PNack* nack)
{
  return nack->lpl3.nackReason;
}

/** \brief Turn an Interest into a Nack.
 *  \param[inout] pkt the packet, must be Interest
 */
void MakeNack(struct rte_mbuf* pkt, NackReason reason);

#endif // NDN_DPDK_NDN_NACK_H
