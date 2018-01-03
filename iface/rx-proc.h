#ifndef NDN_DPDK_IFACE_RX_PROC_H
#define NDN_DPDK_IFACE_RX_PROC_H

#include "counters.h"
#include "in-order-reassembler.h"

/// \file

/** \brief Incoming frame processing procedure.
 */
typedef struct RxProc
{
  InOrderReassembler reassembler;

  uint64_t nFrames; ///< input frames
  uint64_t nOctets; ///< input bytes

  uint64_t nInterests; ///< decoded Interests
  uint64_t nData;      ///< decoded Data
  uint64_t nNacks;     ///< decoded Nacks

  uint64_t nL2DecodeErr; ///< failed NDNLP decodings
  uint64_t nL3DecodeErr; ///< failed Interest/Data/Nack decodings
} RxProc;

/** \brief Process an incoming L2 frame.
 *  \param pkt incoming L2 frame, starting from NDNLP header;
 *             RxProc retains ownership of this packet
 *  \return L3 packet, with parsed LpPkt and InterestPkt/DataPkt;
 *          RxProc releases ownership of this packet
 *  \retval NULL no L3 packet is ready at this moment
 */
struct rte_mbuf* RxProc_Input(RxProc* rx, struct rte_mbuf* pkt);

/** \brief Retrieve counters.
 */
void RxProc_ReadCounters(RxProc* rx, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_RX_PROC_H
