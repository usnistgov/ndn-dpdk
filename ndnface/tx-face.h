#ifndef NDN_DPDK_NDNFACE_TX_FACE_H
#define NDN_DPDK_NDNFACE_TX_FACE_H

#include "common.h"

/// \file

/** \brief Network interface for transmitting NDN packets.
 */
typedef struct TxFace
{
  uint16_t port;
  uint16_t queue;

  uint16_t mtu;
  struct ether_hdr ethhdr;

  /** \brief number of L2 frames sent, divided by L3 packet type
   *
   *  \li nPkts[NdnPktType_None] idle packets and non-first fragments
   *  \li nPkts[NdnPktType_Interests] Interests
   *  \li nPkts[NdnPktType_Data] Data
   *  \li nPkts[NdnPktType_Nacks] Nacks
   */
  uint64_t nPkts[NdnPktType_MAX];
  uint64_t nOctets; // number of octets sent, including all headers

  void* __txCallback;
} TxFace;

/** \brief Initialize TxFace
 *  \param face the face; port and queue must be assigned
 *  \return whether success
 */
bool TxFace_Init(TxFace* face);

/** \brief Deinitialize TxFace
 *  \param face the face
 */
void TxFace_Close(TxFace* face);

/** \brief Send a burst of packet.
 *  \param face the face
 *  \param pkts array of packet pointers
 *  \param nPkts size of \p pkt array
 *  \return number of sent packets
 */
uint16_t TxFace_TxBurst(TxFace* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_NDNFACE_RX_FACE_H
