#ifndef NDN_DPDK_NDNI_PACKET_H
#define NDN_DPDK_NDNI_PACKET_H

/** @file */

#include "data.h"
#include "nack.h"

const char*
PktType_ToString(PktType t);

/** @brief Convert to parsed packet type. */
static __rte_always_inline PktType
PktType_ToFull(PktType t)
{
  return (t & 0x03);
}

/** @brief Convert to unparsed packet type. */
static __rte_always_inline PktType
PktType_ToSlim(PktType t)
{
  return (t & 0x03) | 0x04;
}

/** @brief mbuf private data for NDN packet. */
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
static_assert(offsetof(PacketPriv, lp) + offsetof(LpHeader, l3) == offsetof(PacketPriv, lpl3), "");
static_assert(offsetof(PacketPriv, nack) + offsetof(PNack, lpl3) == offsetof(PacketPriv, lpl3), "");
static_assert(offsetof(PacketPriv, nack) + offsetof(PNack, interest) ==
                offsetof(PacketPriv, interest),
              "");

/**
 * @brief Convert Packet* from rte_mbuf*.
 * @param pkt mbuf of first fragment; must have sizeof(PacketPriv) priv_size.
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline Packet*
Packet_FromMbuf(struct rte_mbuf* pkt)
{
  assert(pkt->priv_size >= sizeof(PacketPriv));
  return (Packet*)pkt;
}

/** @brief Convert Packet* to rte_mbuf*. */
static __rte_always_inline struct rte_mbuf*
Packet_ToMbuf(const Packet* npkt)
{
  return (struct rte_mbuf*)npkt;
}

/** @brief Get packet type. */
__attribute__((nonnull)) static __rte_always_inline PktType
Packet_GetType(const Packet* npkt)
{
  return Packet_ToMbuf(npkt)->inner_l3_type;
}

/** @brief Set packet type. */
__attribute__((nonnull)) static __rte_always_inline void
Packet_SetType(Packet* npkt, PktType t)
{
  Packet_ToMbuf(npkt)->inner_l3_type = t;
}

__attribute__((nonnull, returns_nonnull)) static __rte_always_inline PacketPriv*
Packet_GetPriv_(Packet* npkt)
{
  return (PacketPriv*)rte_mbuf_to_priv_(rte_mbuf_from_indirect(Packet_ToMbuf(npkt)));
}

/**
 * @brief Access LpHeader* header.
 * @pre Packet_GetType(npkt) == PktFragment
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline LpHeader*
Packet_GetLpHdr(Packet* npkt)
{
  assert(Packet_GetType(npkt) == PktFragment);
  return &Packet_GetPriv_(npkt)->lp;
}

/** @brief Access LpL3* header. */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline LpL3*
Packet_GetLpL3Hdr(Packet* npkt)
{
  return &Packet_GetPriv_(npkt)->lpl3;
}

/**
 * @brief Access PInterest* header.
 * @pre Packet_GetType(npkt) == PktInterest
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline PInterest*
Packet_GetInterestHdr(Packet* npkt)
{
  assert(Packet_GetType(npkt) == PktInterest);
  return &Packet_GetPriv_(npkt)->interest;
}

/**
 * @brief Access PData* header.
 * @pre Packet_GetType(npkt) == PktData
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline PData*
Packet_GetDataHdr(Packet* npkt)
{
  assert(Packet_GetType(npkt) == PktData);
  return &Packet_GetPriv_(npkt)->data;
}

/**
 * @brief Access PNack* header.
 * @pre Packet_GetType(npkt) == PktNack
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline PNack*
Packet_GetNackHdr(Packet* npkt)
{
  assert(Packet_GetType(npkt) == PktNack);
  return &Packet_GetPriv_(npkt)->nack;
}

/**
 * @brief Parse layer 2 and layer 3 in mbuf.
 * @param npkt a uniquely owned, unsegmented, direct mbuf.
 * @return whether success.
 * @post If the packet is fragmented, Packet_GetType(npkt) returns @c PktFragment .
 *       Otherwise, same as @c Packet_ParseL3 .
 */
__attribute__((nonnull, warn_unused_result)) bool
Packet_Parse(Packet* npkt);

/**
 * @brief Parse layer 3 in mbuf.
 * @param npkt a uniquely owned, possibly segmented, direct mbuf.
 *             Its PacketPriv.lpl3 should have been initialized.
 * @return whether success.
 * @post Packet_GetType(npkt) returns @c PktInterest , @c PktData , or @c PktNack .
 * @post If the packet is not fragmented, one of @c PInterest , @c PData , or @c PNack is
 * initialized.
 */
__attribute__((nonnull, warn_unused_result)) bool
Packet_ParseL3(Packet* npkt);

/**
 * @brief Clone packet as indirect mbufs.
 * @retval NULL allocation failure.
 * @return an empty mbuf without PacketPriv, followed by indirect mbufs.
 */
__attribute__((nonnull)) Packet*
Packet_Clone(Packet* npkt, struct rte_mempool* headerMp, struct rte_mempool* indirectMp);

#endif // NDN_DPDK_NDNI_PACKET_H
