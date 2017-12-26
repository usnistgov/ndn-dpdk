#ifndef NDN_DPDK_NDNFACE_TX_FACE_H
#define NDN_DPDK_NDNFACE_TX_FACE_H

#include "common.h"

/// \file

static size_t
TxFace_GetHeaderMempoolDataRoom()
{
  return sizeof(struct ether_hdr) + 1 + 3 + // LpPacket
         1 + 1 + 8 +                        // SeqNo
         1 + 1 + 2 +                        // FragIndex
         1 + 1 + 2 +                        // FragCount
         3 + 1 + 3 + 1 + 1 +                // Nack
         3 + 1 + 1 +                        // CongestionMark
         1 + 3;                             // Payload
}

/** \brief Network interface for transmitting NDN packets.
 */
typedef struct TxFace
{
  uint16_t port;
  uint16_t queue;

  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs

  /** \brief mempool for Ethernet and NDNLP headers
   *
   *  Minimal data room is TxFace_GetHeaderMempoolDataRoom().
   *  There is no requirement on priv size.
   */
  struct rte_mempool* headerMp;

  uint16_t mtu;
  struct ether_hdr ethhdr;

  /** \brief number of L2 frames sent, seperated by L3 packet type
   *
   *  \li nPkts[NdnPktType_None] idle packets and non-first fragments
   *  \li nPkts[NdnPktType_Interests] Interests
   *  \li nPkts[NdnPktType_Data] Data
   *  \li nPkts[NdnPktType_Nacks] Nacks
   */
  uint64_t nPkts[NdnPktType_MAX];
  uint64_t
    nOctets; ///< number of octets sent, including Ethernet and NDNLP headers

  uint64_t
    nAllocFails;    ///< count of bursts that encountered allocation failures
  uint64_t nBursts; ///< count of L2 bursts
  uint64_t nZeroBursts; ///< count of bursts where zero frames were sent
  uint64_t
    nPartialBursts; ///< count of bursts where some (incl all) frames were lost

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
 *
 *  This function creates indirect mbufs to reference \p pkts. The caller may not overwrite \p pkts
 *  while they are being sent, but the caller must free them if no longer needed.
 */
void TxFace_TxBurst(TxFace* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_NDNFACE_RX_FACE_H
