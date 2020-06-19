#ifndef NDN_DPDK_IFACE_RX_PROC_H
#define NDN_DPDK_IFACE_RX_PROC_H

/// \file

#include "in-order-reassembler.h"

#define RXPROC_MAX_THREADS 8

/** \brief RxProc per-thread information.
 */
typedef struct RxProcThread
{
  /** \brief input frames and decoded L3 packets
   *
   *  \li nFrames[L3PktTypeNone] input frames
   *  \li nFrames[L3PktTypeInterests] decoded Interests
   *  \li nFrames[L3PktTypeData] decoded Data
   *  \li nFrames[L3PktTypeNacks] decoded Nacks
   */
  uint64_t nFrames[L3PktTypeMAX];
  uint64_t nOctets; ///< input bytes

  uint64_t nL2DecodeErr; ///< failed NDNLP decodings
  uint64_t nL3DecodeErr; ///< failed Interest/Data/Nack decodings
} __rte_cache_aligned RxProcThread;

/** \brief Incoming frame processing procedure.
 */
typedef struct RxProc
{
  struct rte_mempool* nameMp; ///< mempool for allocating Name linearize mbufs

  InOrderReassembler reassembler;

  RxProcThread threads[RXPROC_MAX_THREADS];
} RxProc;

/** \brief Initialize RX procedure.
 *  \pre *rx is zeroized.
 *  \param nameMp mempool for name linearize; dataroom must be at least NameMaxLength.
 */
int
RxProc_Init(RxProc* rx, struct rte_mempool* nameMp);

/** \brief Process an incoming L2 frame.
 *  \param pkt incoming L2 frame, starting from NDNLP header;
 *             RxProc retains ownership of this packet
 *  \return L3 packet after \c Packet_ParseL3;
 *          RxProc releases ownership of this packet
 *  \retval NULL no L3 packet is ready at this moment
 */
Packet*
RxProc_Input(RxProc* rx, int thread, struct rte_mbuf* pkt);

#endif // NDN_DPDK_IFACE_RX_PROC_H
