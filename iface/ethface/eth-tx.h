#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_TX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_TX_H

#include "common.h"

/// \file

typedef struct EthFace EthFace;

static size_t
EthTx_GetHeaderMempoolDataRoom()
{
  return sizeof(struct ether_hdr) + EncodeLpHeaders_GetHeadroom() +
         EncodeLpHeaders_GetTailroom();
}

typedef struct EthTx
{
  uint16_t queue;

  struct rte_mempool* indirectMp;
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
} __rte_cache_aligned EthTx;

int EthTx_Init(EthFace* face, EthTx* tx);

void EthTx_Close(EthFace* face, EthTx* tx);

void EthTx_TxBurst(EthFace* face, EthTx* tx, struct rte_mbuf** pkts,
                   uint16_t nPkts);

void EthTx_ReadCounters(EthFace* face, EthTx* tx, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_TX_H