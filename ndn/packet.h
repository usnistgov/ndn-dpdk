#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

#include "data-pkt.h"
#include "interest-pkt.h"
#include "lp-pkt.h"

/// \file

/** \brief Indicate layer 2 packet type.
 *
 *  L2PktType is stored in rte_mbuf.inner_l2_type field.
 *  It reflects what is stored in \p PacketPriv of direct mbuf.
 */
typedef enum L2PktType {
  L2PktType_None,
  L2PktType_NdnlpV2,
} L2PktType;

static inline L2PktType
Packet_GetL2PktType(const struct rte_mbuf* pkt)
{
  return pkt->inner_l2_type;
}

static inline void
Packet_SetL2PktType(struct rte_mbuf* pkt, L2PktType t)
{
  pkt->inner_l2_type = t;
}

/** \brief Indicate network layer packet type.
 *
 *  NdnPktType is stored in rte_mbuf.inner_l3_type field.
 *  It reflects what is stored in \p PacketPriv of direct mbuf.
 */
typedef enum NdnPktType {
  NdnPktType_None,
  NdnPktType_Interest,
  NdnPktType_Data,
  NdnPktType_Nack,
  NdnPktType_MAX = NdnPktType_Nack
} NdnPktType;

/** \brief Get NDN network layer packet type.
 */
static inline NdnPktType
Packet_GetNdnPktType(const struct rte_mbuf* pkt)
{
  return pkt->inner_l3_type;
}

/** \brief Set NDN network layer packet type.
 */
static inline void
Packet_SetNdnPktType(struct rte_mbuf* pkt, NdnPktType t)
{
  pkt->inner_l3_type = t;
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
  assert(Packet_GetL2PktType(pkt) == L2PktType_NdnlpV2);
  return MbufDirectPriv(pkt, LpPkt*, offsetof(PacketPriv, lp));
}

/** \brief Access InterestPkt* header.
 */
static inline InterestPkt*
Packet_GetInterestHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Interest);
  return MbufDirectPriv(pkt, InterestPkt*, offsetof(PacketPriv, interest));
}

/** \brief Access DataPkt* header
 */
static inline DataPkt*
Packet_GetDataHdr(struct rte_mbuf* pkt)
{
  assert(Packet_GetNdnPktType(pkt) == NdnPktType_Data);
  return MbufDirectPriv(pkt, DataPkt*, offsetof(PacketPriv, data));
}

#endif // NDN_DPDK_NDN_PACKET_H
