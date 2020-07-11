#ifndef NDN_DPDK_IFACE_IN_ORDER_REASSEMBLER_H
#define NDN_DPDK_IFACE_IN_ORDER_REASSEMBLER_H

/** @file */

#include "common.h"

/** @brief Reassembler that requires in-order packet arrival. */
typedef struct InOrderReassembler
{
  struct rte_mbuf* head; ///< first fragment of current packet
  struct rte_mbuf* tail; ///< last fragment of current packet
  uint64_t nextSeqNo;    ///< next expected sequence number; valid iff .tail!=NULL

  uint64_t nAccepted;   ///< number of fragments received and accepted
  uint64_t nOutOfOrder; ///< number of out-of-order fragments dropped
  uint64_t nDelivered;  ///< number of L3 packets delivered
  uint64_t nIncomplete; ///< number of incomplete L3 packets discarded
} InOrderReassembler;

/**
 * @brief Receive an NDNLPv2 fragmented packet into the reassembler.
 * @param npkt the packet after @c Packet_ParseL2; its mbuf must point to LpPayload,
 *             and @c Packet_GetLpHdr must be available.
 * @return reassembled packet, or NULL if still waiting for more fragments.
 */
Packet*
InOrderReassembler_Receive(InOrderReassembler* r, Packet* npkt);

#endif // NDN_DPDK_IFACE_IN_ORDER_REASSEMBLER_H
