#ifndef NDNDPDK_IFACE_FACE_IMPL_H
#define NDNDPDK_IFACE_FACE_IMPL_H

/** @file */

#include "face.h"

/**
 * @brief Process a burst of received L2 frames.
 * @see @c Face_RxInputFunc
 */
__attribute__((nonnull)) void
FaceRx_Input(Face* face, int rxThread, FaceRxInputCtx* ctx);
;

__attribute__((nonnull)) static __rte_always_inline void
FaceTx_CheckDirectFragmentMbuf_(struct rte_mbuf* pkt) {
  NDNDPDK_ASSERT(pkt->pkt_len > 0);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt));
  NDNDPDK_ASSERT(rte_mbuf_refcnt_read(pkt) == 1);
  NDNDPDK_ASSERT(rte_pktmbuf_headroom(pkt) >= RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);
}

/**
 * @brief Process an outgoing L3 packet.
 * @param npkt outgoing L3 packet; face takes ownership.
 * @param[out] frames L2 frames to be transmitted; face releases ownership.
 * @return number of L2 frames to be transmitted.
 */
typedef uint16_t (*FaceTx_OutputFunc)(Face* face, int txThread, Packet* npkt,
                                      struct rte_mbuf* frames[LpMaxFragments]);

/** @brief @c FaceTx_OutputFunc for @c PacketTxAlign.linearize==true with single-segment packet. */
__attribute__((nonnull)) uint16_t
FaceTx_LinearOne(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]);

/** @brief @c FaceTx_OutputFunc for @c PacketTxAlign.linearize==false with single-segment packet. */
__attribute__((nonnull)) uint16_t
FaceTx_ChainedOne(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]);

/** @brief @c FaceTx_OutputFunc for @c PacketTxAlign.linearize==true with multi-segment packet. */
__attribute__((nonnull)) uint16_t
FaceTx_LinearFrag(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]);

/** @brief @c FaceTx_OutputFunc for @c PacketTxAlign.linearize==false with multi-segment packet. */
__attribute__((nonnull)) uint16_t
FaceTx_ChainedFrag(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]);

#endif // NDNDPDK_IFACE_FACE_IMPL_H
