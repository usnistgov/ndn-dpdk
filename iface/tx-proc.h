#ifndef NDN_DPDK_IFACE_TX_PROC_H
#define NDN_DPDK_IFACE_TX_PROC_H

/// \file

#include "counters.h"

typedef struct TxProc TxProc;

typedef uint16_t (*__TxProc_OutputFunc)(TxProc* tx, struct rte_mbuf* pkt,
                                        struct rte_mbuf** frames,
                                        uint16_t maxFrames);

/** \brief Outgoing packet processing procedure.
 */
typedef struct TxProc
{
  struct rte_mempool* indirectMp;
  struct rte_mempool* headerMp;
  __TxProc_OutputFunc outputFunc;

  uint16_t headerHeadroom;      ///< headroom for header mbuf
  uint16_t fragmentPayloadSize; ///< max payload size per fragment

  uint64_t lastSeqNo; ///< last used NDNLP sequence number

  /** \brief number of L3 packets to be queued
   *
   *  \li nL3Pkts[0] packets that did not require fragmentation
   *  \li nL3Pkts[1] packets that required fragmentation
   */
  uint64_t nL3Pkts[2];
  uint64_t nL3OverLength; ///< dropped L3 packets due to over length
  uint64_t nAllocFails;   ///< dropped L3 packets due to allocation failure

  uint64_t nQueueAccepts; ///< number of L2 frames accepted by queue
  uint64_t nQueueRejects; ///< dropped L2 frames due to full queue

  /** \brief number of L2 frames sent, seperated by L3 packet type
   *
   *  \li nFrames[NdnPktType_None] idle packets and non-first fragments
   *  \li nFrames[NdnPktType_Interests] Interests
   *  \li nFrames[NdnPktType_Data] Data
   *  \li nFrames[NdnPktType_Nacks] Nacks
   */
  uint64_t nFrames[NdnPktType_MAX];
  uint64_t nOctets; ///< octets sent, including Ethernet and NDNLP headers
} TxProc;

/** \brief Initialize TX procedure.
 *  \param mtu transport MTU available for NDNLP packets
 *  \param headroom headroom before NDNLP header, as required by transport
 *  \param indirectMp mempool for indirect mbufs
 *  \param headerMp mempool for NDNLP headers; dataroom must be at least headroom +
 *                  EncodeLpHeaders_GetHeadroom() + EncodeLpHeaders_GetTailroom()
 *  \retval 0 success
 *  \retval ENOSPC MTU is too small
 *  \retval ERANGE dataroom of headerMp is too small
 */
int TxProc_Init(TxProc* tx, uint16_t mtu, uint16_t headroom,
                struct rte_mempool* indirectMp, struct rte_mempool* headerMp);

/** \brief Process an outgoing L3 packet.
 *  \param pkt outgoing L3 packet;
 *             TxProc does not retain ownership of this packet
 *  \param[out] frames L2 frames to be transmitted;
 *                     TxProc releases ownership of these frames
 *  \param maxFrames size of frames array
 *  \return number of L2 frames to be transmitted
 */
static inline uint16_t
TxProc_Output(TxProc* tx, struct rte_mbuf* pkt, struct rte_mbuf** frames,
              uint16_t maxFrames)
{
  return (*tx->outputFunc)(tx, pkt, frames, maxFrames);
}

static inline void
TxProc_CountQueued(TxProc* tx, uint16_t nAccepts, uint16_t nRejects)
{
  tx->nQueueAccepts += nAccepts;
  tx->nQueueRejects += nRejects;
}

/** \brief Update counters when L2 frames have been transmitted.
 */
static inline void
TxProc_CountSent(TxProc* tx, struct rte_mbuf* pkt)
{
  ++tx->nFrames[Packet_GetNdnPktType(pkt)];
  tx->nOctets += pkt->pkt_len;
}

/** \brief Retrieve counters.
 */
void TxProc_ReadCounters(TxProc* tx, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_TX_PROC_H
