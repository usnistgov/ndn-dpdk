#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

/// \file

#include "data-pkt.h"
#include "interest-pkt.h"
#include "lp-pkt.h"

/** \brief An NDN packet.
 */
typedef struct
{
} Packet;

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

/** \brief Convert Packet* from rte_mbuf*.
 *  \param pkt mbuf of first fragment; must have sizeof(PacketPriv) privSize.
 */
static Packet*
Packet_FromMbuf(struct rte_mbuf* pkt)
{
  assert(pkt->priv_size >= sizeof(PacketPriv));
  return (Packet*)pkt;
}

/** \brief Convert Packet* to rte_mbuf*.
 */
static struct rte_mbuf*
Packet_ToMbuf(const Packet* npkt)
{
  return (struct rte_mbuf*)npkt;
}

/** \brief Indicate layer 2 packet type.
 *
 *  L2PktType is stored in rte_mbuf.inner_l2_type field.
 */
typedef enum L2PktType {
  L2PktType_None,
  L2PktType_NdnlpV2,
} L2PktType;

static L2PktType
Packet_GetL2PktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l2_type;
}

static void
Packet_SetL2PktType(Packet* npkt, L2PktType t)
{
  Packet_ToMbuf(npkt)->inner_l2_type = t;
}

/** \brief Indicate network layer packet type.
 *
 *  NdnPktType is stored in rte_mbuf.inner_l3_type field.
 */
typedef enum NdnPktType {
  NdnPktType_None,
  NdnPktType_Interest,
  NdnPktType_Data,
  NdnPktType_Nack,
  NdnPktType_MAX
} NdnPktType;

/** \brief Get NDN network layer packet type.
 */
static NdnPktType
Packet_GetNdnPktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l3_type;
}

/** \brief Set NDN network layer packet type.
 */
static void
Packet_SetNdnPktType(Packet* npkt, NdnPktType t)
{
  Packet_ToMbuf(npkt)->inner_l3_type = t;
}

static LpPkt*
Packet_GetLpHdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktType_NdnlpV2);
  return MbufDirectPriv(Packet_ToMbuf(npkt), LpPkt*, offsetof(PacketPriv, lp));
}

/** \brief Access InterestPkt* header.
 */
static InterestPkt*
Packet_GetInterestHdr(Packet* npkt)
{
  assert(Packet_GetNdnPktType(npkt) == NdnPktType_Interest ||
         (Packet_GetNdnPktType(npkt) == NdnPktType_Nack &&
          Packet_GetLpHdr(npkt)->nackReason > 0));
  return MbufDirectPriv(Packet_ToMbuf(npkt), InterestPkt*,
                        offsetof(PacketPriv, interest));
}

/** \brief Access DataPkt* header
 */
static DataPkt*
Packet_GetDataHdr(Packet* npkt)
{
  assert(Packet_GetNdnPktType(npkt) == NdnPktType_Data);
  return MbufDirectPriv(Packet_ToMbuf(npkt), DataPkt*,
                        offsetof(PacketPriv, data));
}

#endif // NDN_DPDK_NDN_PACKET_H
