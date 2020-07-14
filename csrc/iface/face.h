#ifndef NDNDPDK_IFACE_FACE_H
#define NDNDPDK_IFACE_FACE_H

/** @file */

#include "faceid.h"
#include "rx-proc.h"
#include "tx-proc.h"

#include "../core/urcu.h"
#include <urcu/rcuhlist.h>

typedef struct Face Face;
typedef struct FaceCounters FaceCounters;

/**
 * @brief Transmit a burst of L2 frames.
 * @param pkts L2 frames
 * @return successfully queued frames
 * @post FaceImpl owns queued frames, but does not own remaining frames
 */
typedef uint16_t (*FaceImpl_TxBurst)(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

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
  FaceImpl_TxBurst txBurstOp;
  FaceID id;
  FaceState state;

  struct rte_ring* txQueue;
  struct cds_hlist_node txlNode;
} __rte_cache_aligned Face;

static inline void*
Face_GetPriv(Face* face)
{
  return face->impl->priv;
}

#define Face_GetPrivT(face, T) ((T*)Face_GetPriv((face)))

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

/**
 * @brief Send a burst of packets.
 * @param npkts array of L3 packets; face takes ownership
 * @param count size of @p npkts array
 *
 * This function is thread-safe.
 */
static inline void
Face_TxBurst(FaceID faceID, Packet** npkts, uint16_t count)
{
  Face* face = Face_Get(faceID);
  if (unlikely(face->state != FaceStateUp)) {
    rte_pktmbuf_free_bulk_((struct rte_mbuf**)npkts, count);
    return;
  }

  uint16_t nQueued = rte_ring_enqueue_burst(face->txQueue, (void**)npkts, count, NULL);
  uint16_t nRejects = count - nQueued;
  rte_pktmbuf_free_bulk_((struct rte_mbuf**)&npkts[nQueued], nRejects);
  // TODO count nRejects
}

/**
 * @brief Send a packet.
 * @param npkt an L3 packet; face takes ownership
 */
static inline void
Face_Tx(FaceID faceID, Packet* npkt)
{
  Face_TxBurst(faceID, &npkt, 1);
}

#endif // NDNDPDK_IFACE_FACE_H
