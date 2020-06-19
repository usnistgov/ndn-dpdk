#ifndef NDN_DPDK_NDN_PACKET_H
#define NDN_DPDK_NDN_PACKET_H

/// \file

#include "data.h"
#include "interest.h"
#include "lp.h"
#include "nack.h"

/** \brief Get \p t as lower case string.
 */
const char*
L3PktTypeToString(L3PktType t);

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
static inline Packet*
Packet_FromMbuf(struct rte_mbuf* pkt)
{
  assert(pkt->priv_size >= sizeof(PacketPriv));
  return (Packet*)pkt;
}

/** \brief Convert Packet* to rte_mbuf*.
 */
static inline struct rte_mbuf*
Packet_ToMbuf(const Packet* npkt)
{
  return (struct rte_mbuf*)npkt;
}

/** \brief Get layer 2 packet type.
 */
static inline L2PktType
Packet_GetL2PktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l2_type;
}

/** \brief Set layer 2 packet type.
 */
static inline void
Packet_SetL2PktType(Packet* npkt, L2PktType t)
{
  Packet_ToMbuf(npkt)->inner_l2_type = t;
}

/** \brief Get layer 3 packet type.
 */
static inline L3PktType
Packet_GetL3PktType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l3_type;
}

/** \brief Set layer 3 packet type.
 */
static inline void
Packet_SetL3PktType(Packet* npkt, L3PktType t)
{
  Packet_ToMbuf(npkt)->inner_l3_type = t;
}

static inline PacketPriv*
Packet_GetPriv_(Packet* npkt)
{
  return (PacketPriv*)rte_mbuf_to_priv_(
    rte_mbuf_from_indirect(Packet_ToMbuf(npkt)));
}

static inline LpHeader*
Packet_GetLpHdr_(Packet* npkt)
{
  return &Packet_GetPriv_(npkt)->lp;
}

/** \brief Access LpHeader* header.
 */
static inline LpHeader*
Packet_GetLpHdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktTypeNdnlpV2 &&
         Packet_GetL3PktType(npkt) == L3PktTypeNone);
  return Packet_GetLpHdr_(npkt);
}

static inline LpL3*
Packet_GetLpL3Hdr_(Packet* npkt)
{
  return &Packet_GetPriv_(npkt)->lpl3;
}

/** \brief Access LpL3* header.
 */
static inline LpL3*
Packet_GetLpL3Hdr(Packet* npkt)
{
  assert(Packet_GetL2PktType(npkt) == L2PktTypeNdnlpV2);
  return Packet_GetLpL3Hdr_(npkt);
}

/** \brief Access LpL3* header, initialize it if it does not exist.
 */
static inline LpL3*
Packet_InitLpL3Hdr(Packet* npkt)
{
  LpL3* lpl3 = Packet_GetLpL3Hdr_(npkt);
  if (Packet_GetL2PktType(npkt) != L2PktTypeNdnlpV2) {
    Packet_SetL2PktType(npkt, L2PktTypeNdnlpV2);
    memset(lpl3, 0, sizeof(*lpl3));
  }
  return lpl3;
}

static inline PInterest*
Packet_GetInterestHdr_(Packet* npkt)
{
  return &Packet_GetPriv_(npkt)->interest;
}

/** \brief Access PInterest* header.
 */
static inline PInterest*
Packet_GetInterestHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktTypeInterest &&
         (Packet_GetL2PktType(npkt) != L2PktTypeNdnlpV2 ||
          Packet_GetLpL3Hdr_(npkt)->nackReason == NackNone));
  return Packet_GetInterestHdr_(npkt);
}

static inline PData*
Packet_GetDataHdr_(Packet* npkt)
{
  return &Packet_GetPriv_(npkt)->data;
}

/** \brief Access PData* header
 */
static inline PData*
Packet_GetDataHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktTypeData);
  return Packet_GetDataHdr_(npkt);
}

/** \brief Access PNack* header.
 */
static inline PNack*
Packet_GetNackHdr(Packet* npkt)
{
  assert(Packet_GetL3PktType(npkt) == L3PktTypeNack &&
         Packet_GetLpL3Hdr(npkt)->nackReason != NackNone);
  return &Packet_GetPriv_(npkt)->nack;
}

/** \brief Parse packet as LpPacket (including bare Interest/Data).
 *  \retval NdnErrBadType packet type is not LpPacket.
 *  \post Packet_GetL2Type(npkt) == L2PktTypeNdnlpV2
 *  \post LpHeader is stripped, leaving payload TLV-VALUE in the packet.
 */
NdnError
Packet_ParseL2(Packet* npkt);

/** \brief Parse packet as Interest or Data.
 *  \param nameMp mempool for allocating Name linearize mbufs,
 *                requires at least \c NameMaxLength dataroom;
 *                if NULL, will panic when Name linearize becomes necessary.
 *  \retval NdnErrBadType packet type is neither Interest nor Data.
 *  \retval NdnErrAllocError unable to allocate mbuf.
 *  \post Packet_GetL3Type(npkt) is L3PktTypeInterest or L3PktTypeData or L3PktTypeNack.
 */
NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* nameMp);

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
