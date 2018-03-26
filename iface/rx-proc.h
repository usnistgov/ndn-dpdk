#ifndef NDN_DPDK_IFACE_RX_PROC_H
#define NDN_DPDK_IFACE_RX_PROC_H

/// \file

#include "counters.h"
#include "in-order-reassembler.h"

/** \brief Incoming frame processing procedure.
 */
typedef struct RxProc
{
  struct rte_mempool* nameMp; ///< mempool for allocating Name linearize mbufs

  InOrderReassembler reassembler;

  /** \brief input frames and decoded L3 packets
   *
   *  \li nFrames[L3PktType_None] input frames
   *  \li nFrames[L3PktType_Interests] decoded Interests
   *  \li nFrames[L3PktType_Data] decoded Data
   *  \li nFrames[L3PktType_Nacks] decoded Nacks
   */
  uint64_t nFrames[L3PktType_MAX];
  uint64_t nOctets; ///< input bytes

  uint64_t nL2DecodeErr; ///< failed NDNLP decodings
  uint64_t nL3DecodeErr; ///< failed Interest/Data/Nack decodings
} RxProc;

/** \brief Initialize RX procedure.
 *  \param nameMp mempool for name linearize; dataroom must be at least NAME_MAX_LENGTH.
 *  \retval 0
 */
int RxProc_Init(RxProc* rx, struct rte_mempool* nameMp);

/** \brief Process an incoming L2 frame.
 *  \param pkt incoming L2 frame, starting from NDNLP header;
 *             RxProc retains ownership of this packet
 *  \return L3 packet after \c Packet_ParseL3;
 *          RxProc releases ownership of this packet
 *  \retval NULL no L3 packet is ready at this moment
 */
Packet* RxProc_Input(RxProc* rx, struct rte_mbuf* pkt);

/** \brief Retrieve counters.
 */
void RxProc_ReadCounters(RxProc* rx, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_RX_PROC_H
