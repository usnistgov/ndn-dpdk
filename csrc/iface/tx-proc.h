#ifndef NDNDPDK_IFACE_TX_PROC_H
#define NDNDPDK_IFACE_TX_PROC_H

/** @file */

#include "../pdump/face.h"
#include "common.h"

/**
 * @brief Transmit a burst of L2 frames.
 * @param pkts L2 frames
 * @return successfully queued frames
 * @post FaceImpl owns queued frames, but does not own remaining frames
 */
typedef uint16_t (*Face_L2TxBurst)(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

typedef struct TxProc TxProc;

typedef uint16_t (*TxProc_OutputFunc_)(TxProc* tx, Packet* npkt,
                                       struct rte_mbuf* frames[LpMaxFragments],
                                       PacketTxAlign align);

/** @brief Outgoing packet processing procedure. */
typedef struct TxProc
{
  Face_L2TxBurst l2Burst;
  PdumpFaceRef pdump;

  PacketMempools mp; ///< mempools for fragmentation
  TxProc_OutputFunc_ outputFunc[2];
  uint64_t nextSeqNum; ///< next fragmentation sequence number

  uint64_t nL3Fragmented; ///< L3 packets that required fragmentation
  uint64_t nL3OverLength; ///< dropped L3 packets due to over length
  uint64_t nAllocFails;   ///< dropped L3 packets due to allocation failure

  uint64_t nFrames[PktMax]; ///< sent+dropped L2 frames and L3 packets
  uint64_t nOctets;         ///< sent+dropped L2 octets (including LpHeader)
  uint64_t nDroppedFrames;  ///< dropped L2 frames
  uint64_t nDroppedOctets;  ///< dropped L2 octets
} __rte_cache_aligned TxProc;

__attribute__((nonnull)) static __rte_always_inline void
TxProc_CheckDirectFragmentMbuf_(struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(pkt->pkt_len > 0);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt));
  NDNDPDK_ASSERT(rte_mbuf_refcnt_read(pkt) == 1);
  NDNDPDK_ASSERT(rte_pktmbuf_headroom(pkt) >= RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);
}

/**
 * @brief Process an outgoing L3 packet.
 * @param npkt outgoing L3 packet; TxProc takes ownership.
 * @param[out] frames L2 frames to be transmitted; TxProc releases ownership.
 * @return number of L2 frames to be transmitted.
 */
__attribute__((nonnull)) static inline uint16_t
TxProc_Output(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments],
              PacketTxAlign align)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  TxProc_CheckDirectFragmentMbuf_(pkt);

  bool isOneFragment = pkt->pkt_len <= align.fragmentPayloadSize;
  return (tx->outputFunc[(int)isOneFragment])(tx, npkt, frames, align);
}

__attribute__((nonnull)) void
TxProc_Init(TxProc* tx, PacketTxAlign align);

#endif // NDNDPDK_IFACE_TX_PROC_H
