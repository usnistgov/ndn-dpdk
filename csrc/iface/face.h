#ifndef NDNDPDK_IFACE_FACE_H
#define NDNDPDK_IFACE_FACE_H

/** @file */

#include "faceid.h"
#include "rx-proc.h"
#include "tx-proc.h"

#include "../core/urcu.h"
#include <urcu/rcuhlist.h>

typedef struct FaceImpl
{
  RxProc rx;
  TxProc tx;
  char priv[0];
} FaceImpl;

/** @brief Generic network interface. */
typedef struct Face
{
  FaceImpl* impl;
  struct rte_ring* outputQueue;
  struct cds_hlist_node txlNode;
  PacketTxAlign txAlign;
  FaceState state;
  FaceID id;
} __rte_cache_aligned Face;

static inline void*
Face_GetPriv(Face* face)
{
  return face->impl->priv;
}

/** @brief Static array of all faces. */
extern Face gFaces[];

static inline Face*
Face_Get(FaceID id)
{
  return &gFaces[id];
}

/** @brief Return whether the face is DOWN. */
static inline bool
Face_IsDown(FaceID faceID)
{
  Face* face = Face_Get(faceID);
  return face->state != FaceStateUp;
}

static inline PacketTxAlign
Face_PacketTxAlign(FaceID faceID)
{
  Face* face = Face_Get(faceID);
  return face->txAlign;
}

/**
 * @brief Enqueue a burst of packets on the output queue to be transmitted by the output thread.
 * @param npkts array of L3 packets; face takes ownership.
 * @param count size of @p npkts array.
 *
 * This function is thread-safe.
 */
__attribute__((nonnull)) static inline void
Face_TxBurst(FaceID faceID, Packet** npkts, uint16_t count)
{
  Face* face = Face_Get(faceID);
  if (unlikely(face->state != FaceStateUp)) {
    rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, count);
    return;
  }

  for (uint16_t i = 0; i < count; ++i) {
    TxProc_CheckDirectFragmentMbuf_(Packet_ToMbuf(npkts[i])); // XXX
  }

  uint16_t nQueued = rte_ring_enqueue_burst(face->outputQueue, (void**)npkts, count, NULL);
  uint16_t nRejects = count - nQueued;
  rte_pktmbuf_free_bulk((struct rte_mbuf**)&npkts[nQueued], nRejects);
  // TODO count nRejects
}

/**
 * @brief Enqueue a packet on the output queue to be transmitted by the output thread.
 * @param npkt an L3 packet; face takes ownership.
 */
__attribute__((nonnull)) static inline void
Face_Tx(FaceID faceID, Packet* npkt)
{
  Face_TxBurst(faceID, &npkt, 1);
}

#endif // NDNDPDK_IFACE_FACE_H
