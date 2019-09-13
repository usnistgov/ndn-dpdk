#ifndef NDN_DPDK_IFACE_FACE_H
#define NDN_DPDK_IFACE_FACE_H

/// \file

#include "faceid.h"
#include "rx-proc.h"
#include "rxburst.h"
#include "tx-proc.h"

#include "../core/running_stat/running-stat.h"
#include "../core/urcu/urcu.h"
#include <urcu/rcuhlist.h>

typedef struct Face Face;
typedef struct FaceCounters FaceCounters;

/** \brief Transmit a burst of L2 frames.
 *  \param pkts L2 frames
 *  \return successfully queued frames
 *  \post FaceImpl owns queued frames, but does not own remaining frames
 */
typedef uint16_t (*FaceImpl_TxBurst)(Face* face,
                                     struct rte_mbuf** pkts,
                                     uint16_t nPkts);

typedef struct FaceImpl
{
  RxProc rx;
  TxProc tx;

  /** \brief Statistics of L3 latency.
   *
   *  Latency counting starts from packet arrival or generation, and ends when
   *  packet is queuing for transmission; this counts per L3 packet.
   */
  RunningStat latencyStat;

  char priv[0];
} FaceImpl;

/** \brief Generic network interface.
 */
typedef struct Face
{
  FaceImpl* impl;
  FaceImpl_TxBurst txBurstOp;
  FaceId id;
  FaceState state;
  int numaSocket;

  struct rte_ring* txQueue;
  struct cds_hlist_node txLoopNode;
} __rte_cache_aligned Face;

static void*
Face_GetPriv(Face* face)
{
  return face->impl->priv;
}

#define Face_GetPrivT(face, T) ((T*)Face_GetPriv((face)))

/** \brief Static array of all faces.
 */
extern Face gFaces_[FACEID_MAX + 1];

static Face*
Face_Get_(FaceId faceId)
{
  return &gFaces_[faceId];
}

// ---- functions invoked by user of face system ----

/** \brief Return whether the face is DOWN.
 */
static bool
Face_IsDown(FaceId faceId)
{
  Face* face = Face_Get_(faceId);
  return face->state != FACESTA_UP;
}

/** \brief Callback upon packet arrival.
 *
 *  Face base type does not directly provide RX function. Each face
 *  implementation shall have an RxLoop function that accepts this callback.
 */
typedef void (*Face_RxCb)(FaceRxBurst* burst, void* cbarg);

/** \brief Send a burst of packets.
 *  \param npkts array of L3 packets; face takes ownership
 *  \param count size of \p npkts array
 *
 *  This function is thread-safe.
 */
static void
Face_TxBurst(FaceId faceId, Packet** npkts, uint16_t count)
{
  Face* face = Face_Get_(faceId);
  if (unlikely(face->state != FACESTA_UP)) {
    FreeMbufs((struct rte_mbuf**)npkts, count);
    return;
  }

  uint16_t nQueued =
    rte_ring_mp_enqueue_burst(face->txQueue, (void**)npkts, count, NULL);
  uint16_t nRejects = count - nQueued;
  FreeMbufs((struct rte_mbuf**)&npkts[nQueued], nRejects);
  // TODO count nRejects
}

/** \brief Send a packet.
 *  \param npkt an L3 packet; face takes ownership
 */
static void
Face_Tx(FaceId faceId, Packet* npkt)
{
  Face_TxBurst(faceId, &npkt, 1);
}

// ---- functions invoked by face implementation ----

/** \brief Process received frames and invoke upper layer callback.
 *  \param burst FaceRxBurst_GetScratch(burst) must contain received frames.
 *               frame->port indicates FaceId, and frame->timestamp should be set.
 *  \param rxThread RX thread number within each face. Threads receiving frames on the
 *                  same face must use distinct numbers to avoid race condition.
 */
void
FaceImpl_RxBurst(FaceRxBurst* burst,
                 uint16_t nFrames,
                 int rxThread,
                 Face_RxCb cb,
                 void* cbarg);

/** \brief Update counters after a frame is transmitted.
 */
static void
FaceImpl_CountSent(Face* face, struct rte_mbuf* pkt)
{
  TxProc_CountSent(&face->impl->tx, pkt);
}

#endif // NDN_DPDK_IFACE_FACE_H
