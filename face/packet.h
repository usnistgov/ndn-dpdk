#ifndef NDN_DPDK_FACE_PACKET_H
#define NDN_DPDK_FACE_PACKET_H

#include "common.h"

/// \file

/** \brief Indicate packet type.
 *
 *  NdnPktType is stored in rte_mbuf.inner_l4_type field.
 *  It reflects what is stored in MbufPriv area.
 */
typedef enum NdnPktType {
  NdnPktType_None,
  NdnPktType_Lp,
  NdnPktType_Interest,
  NdnPktType_Data,
  NdnPktType_Nack
} NdnPktType;

/** \brief Get NDN network layer packet type.
 */
static inline NdnPktType
Packet_GetNdnPktType(const struct rte_mbuf* pkt)
{
  return pkt->inner_l4_type;
}

/** \brief Set NDN network layer packet type.
 */
static inline void
Packet_SetNdnPktType(struct rte_mbuf* pkt, NdnPktType t)
{
  pkt->inner_l4_type = t;
}

/** \brief Access InterestPkt* header.
 */
static inline InterestPkt*
Packet_GetInterestHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Interest);
  return MbufPriv(pkt, InterestPkt*, 0);
}

/** \brief Access DataPkt* header
 */
static inline DataPkt*
Packet_GetDataHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Data);
  return MbufPriv(pkt, DataPkt*, 0);
}

#endif // NDN_DPDK_FACE_PACKET_H
