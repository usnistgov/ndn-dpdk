#ifndef NDN_DPDK_IFACE_ETHFACE_IN_ORDER_REASSEMBLER_H
#define NDN_DPDK_IFACE_ETHFACE_IN_ORDER_REASSEMBLER_H

#include "common.h"

/// \file

/** \brief Reassembler that requires in-order packet arrival.
 */
typedef struct InOrderReassembler
{
  struct rte_mbuf* head; ///< first fragment of current packet
  struct rte_mbuf* tail; ///< last fragment of current packet
  uint64_t nextSeqNo; ///< next expected sequence number; valid iff .tail!=NULL

  uint64_t nAccepted;   ///< number of fragments received and accepted
  uint64_t nOutOfOrder; ///< number of out-of-order fragments dropped
  uint64_t nDelivered;  ///< number of network layer packets delivered
  uint64_t
    nIncomplete; ///< number of incomplete network layer packets discarded
} InOrderReassembler;

/** \brief Receive an LpPkt into the reassembler.
 *  \param pkt the packet; mbuf offset points to network layer packet.
 *         Reassembler will free \p pkt when necessary.
 *  \return reassembled packet, if any.
 */
struct rte_mbuf* InOrderReassembler_Receive(InOrderReassembler* r,
                                            struct rte_mbuf* pkt);

#endif // NDN_DPDK_IFACE_ETHFACE_IN_ORDER_REASSEMBLER_H