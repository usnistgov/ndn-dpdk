#ifndef NDNDPDK_IFACE_FACE_IMPL_H
#define NDNDPDK_IFACE_FACE_IMPL_H

/** @file */

#include "face.h"

/**
 * @brief Process an incoming L2 frame.
 * @param pkt incoming L2 frame, starting from NDNLP header; face takes ownership.
 * @return L3 packet after @c Packet_ParseL3 ; face releases ownership.
 * @retval NULL no L3 packet is ready at this moment.
 */
__attribute__((nonnull)) Packet*
FaceRx_Input(Face* face, int rxThread, struct rte_mbuf* pkt);

__attribute__((nonnull)) static __rte_always_inline void
FaceTx_CheckDirectFragmentMbuf_(struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(pkt->pkt_len > 0);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt));
  NDNDPDK_ASSERT(rte_mbuf_refcnt_read(pkt) == 1);
  NDNDPDK_ASSERT(rte_pktmbuf_headroom(pkt) >= RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);
}

typedef uint16_t (*FaceTx_OutputFunc)(Face* face, int txThread, Packet* npkt,
                                      struct rte_mbuf* frames[LpMaxFragments]);

extern FaceTx_OutputFunc FaceTx_OutputFuncs[];

#define FaceTx_OutputFuncIndex(linear, oneFrag) (((int)(linear) << 1) | ((int)(oneFrag) << 0))

/**
 * @brief Process an outgoing L3 packet.
 * @param npkt outgoing L3 packet; face takes ownership.
 * @param[out] frames L2 frames to be transmitted; face releases ownership.
 * @return number of L2 frames to be transmitted.
 */
__attribute__((nonnull)) static inline uint16_t
FaceTx_Output(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments])
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceTx_CheckDirectFragmentMbuf_(pkt);

  bool isOneFragment = pkt->pkt_len <= face->txAlign.fragmentPayloadSize;
  return FaceTx_OutputFuncs[FaceTx_OutputFuncIndex(face->txAlign.linearize, isOneFragment)](
    face, txThread, npkt, frames);
}

#endif // NDNDPDK_IFACE_FACE_IMPL_H
