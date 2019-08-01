#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

/// \file

#include "data.h"
#include "interest.h"
#include "lp.h"
#include "nack.h"

/** \brief An NDN L2 or L3 packet.
 *
 *  Packet* is struct rte_mbuf* that fulfills requirements of \c Packet_FromMbuf.
 */
typedef struct Packet
{
} Packet;

/** \brief Information stored in rte_mbuf private area.
 */
typedef union PacketPriv
{
  LpHeader lp;
  struct
  {
    LpL3 lpl3;
    union
    {
      PInterest interest;
      PData data;
    };
  };
  PNack nack;
} PacketPriv;
static_assert(offsetof(PacketPriv, lp) + offsetof(LpHeader, l3) ==
                offsetof(PacketPriv, lpl3),
              "");
static_assert(offsetof(PacketPriv, nack) + offsetof(PNack, lpl3) ==
                offsetof(PacketPriv, lpl3),
              "");
static_assert(offsetof(PacketPriv, nack) + offsetof(PNack, interest) ==
                offsetof(PacketPriv, interest),
              "");

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
typedef enum L2PktType
{
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

/** \brief Indicate layer 3 packet type.
 *
 *  L3PktType is stored in rte_mbuf.inner_l3_type field.
 */
typedef enum L3PktType
{
  L3PktType_None,
  L3PktType_Interest,
  L3PktType_Data,
  L3PktType_Nack,
  L3PktType_MAX
} L3PktType;

/** \brief Get \p t as lower case string.
 */
const char*
L3PktType_ToString(L3PktType t);

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

static PacketPriv*
__Packet_GetPriv(Packet* npkt)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return (PacketPriv*)rte_mbuf_to_priv(
    rte_mbuf_from_indirect(Packet_ToMbuf(npkt)));
#pragma GCC diagnostic pop
}

static LpHeader*
__Packet_GetLpHdr(Packet* npkt)
{
  return &__Packet_GetPriv(npkt)->lp;
}

/** \brief Access LpHeader* header.
 */
static LpHeader*
Packet_GetLpHdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktType_NdnlpV2 &&
         Packet_GetL3PktType(npkt) == L3PktType_None);
  return __Packet_GetLpHdr(npkt);
}

static LpL3*
__Packet_GetLpL3Hdr(Packet* npkt)
{
  return &__Packet_GetPriv(npkt)->lpl3;
}

/** \brief Access LpL3* header.
 */
static LpL3*
Packet_GetLpL3Hdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktType_NdnlpV2);
  return __Packet_GetLpL3Hdr(npkt);
}

/** \brief Access LpL3* header, initialize it if it does not exist.
 */
static LpL3*
Packet_InitLpL3Hdr(Packet* npkt)
{
  LpL3* lpl3 = __Packet_GetLpL3Hdr(npkt);
  if (Packet_GetL2PktType(npkt) != L2PktType_NdnlpV2) {
    Packet_SetL2PktType(npkt, L2PktType_NdnlpV2);
    memset(lpl3, 0, sizeof(*lpl3));
  }
  return lpl3;
}

static PInterest*
__Packet_GetInterestHdr(Packet* npkt)
{
  return &__Packet_GetPriv(npkt)->interest;
}

/** \brief Access PInterest* header.
 */
static PInterest*
Packet_GetInterestHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest &&
         (Packet_GetL2PktType(npkt) != L2PktType_NdnlpV2 ||
          __Packet_GetLpL3Hdr(npkt)->nackReason == NackReason_None));
  return __Packet_GetInterestHdr(npkt);
}

static PData*
__Packet_GetDataHdr(Packet* npkt)
{
  return &__Packet_GetPriv(npkt)->data;
}

/** \brief Access PData* header
 */
static PData*
Packet_GetDataHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Data);
  return __Packet_GetDataHdr(npkt);
}

/** \brief Access PNack* header.
 */
static PNack*
Packet_GetNackHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Nack &&
         Packet_GetLpL3Hdr(npkt)->nackReason != NackReason_None);
  return &__Packet_GetPriv(npkt)->nack;
}

/** \brief Parse packet as LpPacket (including bare Interest/Data).
 *  \retval NdnError_BadType packet type is not LpPacket.
 *  \post Packet_GetL2Type(npkt) == L2PktType_NdnlpV2
 *  \post LpHeader is stripped, leaving payload TLV-VALUE in the packet.
 */
NdnError
Packet_ParseL2(Packet* npkt);

/** \brief Parse packet as Interest or Data.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \c NAME_MAX_LENGTH dataroom;
 *                if NULL, will panic when Name linearize becomes necessary.
 *  \retval NdnError_BadType packet type is neither Interest nor Data.
 *  \retval NdnError_AllocError unable to allocate mbuf.
 *  \post Packet_GetL3Type(npkt) is L3PktType_Interest or L3PktType_Data or L3PktType_Nack.
 */
NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* nameMp);

/** \brief Copy timestamp from \p src to \p dst.
 */
static void
Packet_CopyTimestamp(Packet* dst, Packet* src)
{
  Packet_ToMbuf(dst)->timestamp = Packet_ToMbuf(src)->timestamp;
}

/** \brief Clone packet with a new empty header mbuf and indirect mbufs.
 *  \param[in] npkt the original packet.
 *  \param headerMp mempool for header mbuf;
 *                  must fulfill requirements of \c Packet_FromMbuf();
 *                  may have additional headroom for lower layer headers.
 *  \param indirectMp mempool for allocating indirect mbufs.
 *  \return cloned packet with copied PacketPriv.
 *  \retval NULL allocation failure.
 */
Packet*
ClonePacket(Packet* npkt,
            struct rte_mempool* headerMp,
            struct rte_mempool* indirectMp);

#endif // NDN_DPDK_NDN_PACKET_H
