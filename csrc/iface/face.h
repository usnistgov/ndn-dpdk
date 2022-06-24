#ifndef NDNDPDK_IFACE_FACE_H
#define NDNDPDK_IFACE_FACE_H

/** @file */

#include "input-demux.h"
#include "reassembler.h"

#include "../core/urcu.h"
#include "../pdump/source.h"
#include <urcu/rcuhlist.h>

/** @brief Face RX per-thread information. */
typedef struct FaceRxThread
{
  uint64_t nFrames[PktMax]; ///< accepted L3 packets; nFrames[0] is nOctets
  uint64_t nDecodeErr;      ///< decode errors
  Reassembler reass;
} __rte_cache_aligned FaceRxThread;

/** @brief Face TX per-thread information. */
typedef struct FaceTxThread
{
  uint64_t nextSeqNum; ///< next fragmentation sequence number

  uint64_t nL3Fragmented; ///< L3 packets that required fragmentation
  uint64_t nL3OverLength; ///< dropped L3 packets due to over length
  uint64_t nAllocFails;   ///< dropped L3 packets due to allocation failure

  uint64_t nFrames[PktMax]; ///< sent+dropped L2 frames and L3 packets
  uint64_t nOctets;         ///< sent+dropped L2 octets (including LpHeader)
  uint64_t nDroppedFrames;  ///< dropped L2 frames
  uint64_t nDroppedOctets;  ///< dropped L2 octets
} __rte_cache_aligned FaceTxThread;

/**
 * @brief Transmit a burst of L2 frames.
 * @param pkts L2 frames.
 * @return successfully queued frames.
 * @post FaceImpl owns queued frames, but does not own remaining frames.
 */
typedef uint16_t (*Face_TxBurstFunc)(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

/**
 * @brief Face details.
 *
 * Fields in this struct are meant to be accessed in RX and TX threads, but not in
 * forwarding/producer/consumer threads.
 */
typedef struct FaceImpl
{
  FaceRxThread rx[MaxFaceRxThreads];
  FaceTxThread tx[MaxFaceTxThreads];

  InputDemuxes* rxDemuxes; ///< per-face demuxes, overrides RxLoop demuxes
  PdumpSourceRef rxPdump;

  PacketMempools txMempools; ///< mempools for fragmentation
  Face_TxBurstFunc txBurst;
  PdumpSourceRef txPdump;

  uint8_t priv[] __rte_cache_aligned;
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
};
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
