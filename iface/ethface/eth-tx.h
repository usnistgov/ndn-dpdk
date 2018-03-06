#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_TX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_TX_H

/// \file

#include "common.h"

typedef struct EthFace EthFace;

static uint16_t
EthTx_GetHeaderMempoolDataRoom()
{
  return sizeof(struct ether_hdr) + PrependLpHeader_GetHeadroom();
}

/** \brief Ethernet transmitting queue.
 */
typedef struct EthTx
{
  EthFace* face;
  struct ether_hdr ethhdr; // outgoing Ethernet header
  void* txCallback;
} EthTx;

/** \brief Initialize Ethernet TX
 *  \return 0 for success, otherwise error code
 */
int EthTx_Init(EthFace* face, uint16_t queue);

void EthTx_Close(EthFace* face, uint16_t queue);

uint16_t EthTx_TxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
                       uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_TX_H