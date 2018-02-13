#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

/// \file

#include "data.h"
#include "interest-pkt.h"
#include "lp-pkt.h"

/** \brief An NDN L2 or L3 packet.
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
    PData data;
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

/** \brief Get layer 2 packet type.
 */
static L2PktType
Packet_GetL2PktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l2_type;
}

/** \brief Set layer 2 packet type.
 */
static void
Packet_SetL2PktType(Packet* npkt, L2PktType t)
{
  Packet_ToMbuf(npkt)->inner_l2_type = t;
}

static LpPkt*
Packet_GetLpHdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktType_NdnlpV2);
  return MbufDirectPriv(Packet_ToMbuf(npkt), LpPkt*, offsetof(PacketPriv, lp));
}

/** \brief Indicate layer 3 packet type.
 *
 *  L3PktType is stored in rte_mbuf.inner_l3_type field.
 */
typedef enum L3PktType {
  L3PktType_None,
  L3PktType_Interest,
  L3PktType_Data,
  L3PktType_Nack,
  L3PktType_MAX
} L3PktType;

/** \brief Get layer 3 packet type.
 */
static L3PktType
Packet_GetL3PktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l3_type;
}

/** \brief Set layer 3 packet type.
 */
static void
Packet_SetL3PktType(Packet* npkt, L3PktType t)
{
  Packet_ToMbuf(npkt)->inner_l3_type = t;
}

/** \brief Parse packet as either Interest or Data.
 *  \param mpName mempool for allocating Name linearize mbufs,
 *                requires at least \p NAME_MAX_LENGTH dataroom.
 *  \retval NdnError_BadType packet is neither Interest nor Data.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 */
NdnError Packet_ParseL3(Packet* npkt, struct rte_mempool* mpName);

/** \brief Access InterestPkt* header.
 */
static InterestPkt*
Packet_GetInterestHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest ||
         (Packet_GetL3PktType(npkt) == L3PktType_Nack &&
          Packet_GetLpHdr(npkt)->nackReason > 0));
  return MbufDirectPriv(Packet_ToMbuf(npkt), InterestPkt*,
                        offsetof(PacketPriv, interest));
}

static PData*
__Packet_GetDataHdr(Packet* npkt)
{
  return MbufDirectPriv(Packet_ToMbuf(npkt), PData*,
                        offsetof(PacketPriv, data));
}

/** \brief Access PData* header
 */
static PData*
Packet_GetDataHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Data);
  return __Packet_GetDataHdr(npkt);
}

#endif // NDN_DPDK_NDN_PACKET_H
