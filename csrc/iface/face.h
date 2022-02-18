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
  uint8_t priv[];
} FaceImpl;

/** @brief Generic network interface. */
struct Face
{
  FaceImpl* impl;
  struct rte_ring* outputQueue;
  struct cds_hlist_node txlNode;
  PacketTxAlign txAlign;
  FaceID id;
  FaceState state;
} __rte_cache_aligned;
static_assert(sizeof(Face) <= RTE_CACHE_LINE_SIZE, "");

__attribute__((nonnull, returns_nonnull)) static inline void*
Face_GetPriv(Face* face)
{
  return face->impl->priv;
}

/** @brief Static array of all faces. */
extern Face gFaces[];

/**
 * @brief Retrieve face by ID.
 *
 * Return value will not be NULL. @c face->impl==NULL indicates non-existent face.
 */
__attribute__((returns_nonnull)) static inline Face*
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

/** @brief Retrieve face TX alignment requirement. */
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
  if (likely(face->state == FaceStateUp)) {
    Mbuf_EnqueueVector((struct rte_mbuf**)npkts, count, face->outputQueue, true);
    // TODO count rejects
  } else {
    rte_pktmbuf_free_bulk((struct rte_mbuf**)npkts, count);
  }
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
