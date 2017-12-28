#ifndef NDN_DPDK_NDNFACE_TX_FACE_H
#define NDN_DPDK_NDNFACE_TX_FACE_H

#include "common.h"

/// \file

static size_t
TxFace_GetHeaderMempoolDataRoom()
{
  return sizeof(struct ether_hdr) + EncodeLpHeaders_GetHeadroom() +
         EncodeLpHeaders_GetTailroom();
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

  /** \brief number of L2 frames sent, seperated by L3 packet type
   *
   *  \li nPkts[NdnPktType_None] idle packets and non-first fragments
   *  \li nPkts[NdnPktType_Interests] Interests
   *  \li nPkts[NdnPktType_Data] Data
   *  \li nPkts[NdnPktType_Nacks] Nacks
   */
  uint64_t nPkts[NdnPktType_MAX];
  uint64_t nOctets; ///< octets sent, including Ethernet and NDNLP headers

  uint64_t nL3Bursts;     ///< total L3 bursts
  uint64_t nL3OverLength; ///< dropped L3 packets due to over length
  uint64_t nAllocFails;   ///< dropped L3 bursts due to allocation failure

  uint64_t nL2Bursts;     ///< total L2 bursts
  uint64_t nL2Incomplete; ///< incomplete L2 bursts due to full queue

  uint16_t fragmentPayloadSize; ///< max payload size per fragment
  struct ether_hdr ethhdr;

  uint64_t lastSeqNo; ///< last used NDNLP sequence number

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
 *  This function creates indirect mbufs to reference \p pkts. The caller may not modify these
 *  packets while they are being sent, but must free them if no longer needed.
 */
void TxFace_TxBurst(TxFace* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_NDNFACE_RX_FACE_H
