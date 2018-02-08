#ifndef NDN_DPDK_NDN_NACK_PKT_H
#define NDN_DPDK_NDN_NACK_PKT_H

/// \file

#include "common.h"

/** \brief Indicate the Nack reason.
 */
typedef enum NackReason {
  NackReason_None = 0, ///< packet is not a Nack
  NackReason_Congestion = 50,
  NackReason_Duplicate = 100,
  NackReason_NoRoute = 150,
  NackReason_Unspecified = 255 ///< reason unspecified
} NackReason;

/** \brief Turn an Interest into a Nack.
 *  \param[inout] pkt the packet, must be Interest
 */
void MakeNack(struct rte_mbuf* pkt, NackReason reason);

#endif // NDN_DPDK_NDN_NACK_PKT_H