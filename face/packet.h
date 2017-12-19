#ifndef NDN_DPDK_FACE_PACKET_H
#define NDN_DPDK_FACE_PACKET_H

#include "common.h"

/// \file

/** \brief Indicate NDN network layer packet type.
 */
typedef enum NdnNetType {
  NdnNetType_None,
  NdnNetType_Interest,
  NdnNetType_Data,
  NdnNetType_Nack
} NdnNetType;

/** \brief Get NDN network layer packet type.
 */
static inline NdnNetType
Packet_GetNdnNetType(const struct rte_mbuf* pkt)
{
  return pkt->inner_l4_type;
}

/** \brief Set NDN network layer packet type.
 */
static inline void
Packet_SetNdnNetType(struct rte_mbuf* pkt, NdnNetType t)
{
  pkt->inner_l4_type = t;
}

/** \brief Access InterestPkt* header.
 */
static inline InterestPkt*
Packet_GetInterestHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnNetType(pkt) == NdnNetType_Interest);
  return MbufPriv(pkt, InterestPkt*, 0);
}

/** \brief Access DataPkt* header
 */
static inline DataPkt*
Packet_GetDataHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnNetType(pkt) == NdnNetType_Data);
  return MbufPriv(pkt, DataPkt*, 0);
}

#endif // NDN_DPDK_FACE_PACKET_H
