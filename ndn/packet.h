#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

#include "data-pkt.h"
#include "interest-pkt.h"
#include "lp-pkt.h"

/// \file

/** \brief Indicate packet type.
 *
 *  NdnPktType is stored in rte_mbuf.inner_l4_type field.
 *  It reflects what is stored in \p PacketPriv.
 */
typedef enum NdnPktType {
  NdnPktType_None,
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

/** \brief Information stored in rte_mbuf private area.
 */
typedef struct PacketPriv
{
  LpPkt lp;
  union
  {
    InterestPkt interest;
    DataPkt data;
  };
} PacketPriv;

static inline LpPkt*
Packet_GetLpHdr(struct rte_mbuf* pkt)
{
  return MbufPriv(pkt, LpPkt*, offsetof(PacketPriv, lp));
}

/** \brief Access InterestPkt* header.
 */
static inline InterestPkt*
Packet_GetInterestHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Interest);
  return MbufPriv(pkt, InterestPkt*, offsetof(PacketPriv, interest));
}

/** \brief Access DataPkt* header
 */
static inline DataPkt*
Packet_GetDataHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Data);
  return MbufPriv(pkt, DataPkt*, offsetof(PacketPriv, data));
}

#endif // NDN_DPDK_NDN_PACKET_H
