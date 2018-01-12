#include "eth-tx.h"
#include "eth-face.h"

#include "../../core/logger.h"
#include "../../dpdk/ethdev.h"

#define LOG_PREFIX "(%" PRIu16 ",%" PRIu16 ") "
#define LOG_PARAM face->port, tx->queue

// max L2 burst size
static const int MAX_FRAMES = 64;

// max fragments per network layer packet
static const int MAX_FRAGMENTS = 16;

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

// callback after NIC transmits packets
static uint16_t
EthTx_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                 uint16_t nPkts, void* ethTxPtr)
{
  EthTx* tx = (EthTx*)ethTxPtr;
  EthFace* face = tx->face;
  assert(face->port == port);
  assert(tx == &face->tx[queue]);

  for (uint16_t i = 0; i < nPkts; ++i) {
    FaceImpl_CountSent(&face->base, pkts[i]);
  }
  return nPkts;
}

int
EthTx_Init(EthFace* face, uint16_t queue)
{
  EthTx* tx = &face->tx[queue];
  tx->face = face;

  EthDev_GetMacAddr(face->port, &tx->ethhdr.s_addr);
  const uint8_t dstAddr[] = { NDN_ETHER_MCAST };
  rte_memcpy(&tx->ethhdr.d_addr, dstAddr, sizeof(tx->ethhdr.d_addr));
  tx->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  tx->txCallback =
    rte_eth_add_tx_callback(face->port, queue, &EthTx_TxCallback, tx);
  if (tx->txCallback == NULL) {
    return rte_errno;
  }

  return 0;
}

void
EthTx_Close(EthFace* face, uint16_t queue)
{
  EthTx* tx = &face->tx[queue];
  rte_eth_remove_tx_callback(face->port, queue, tx->txCallback);
  tx->txCallback = NULL;
}

uint16_t
EthTx_TxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
              uint16_t nPkts)
{
  EthTx* tx = &face->tx[queue];
  for (uint16_t i = 0; i < nPkts; ++i) {
    char* room = rte_pktmbuf_prepend(pkts[i], sizeof(tx->ethhdr));
    assert(room != NULL); // enough headroom is required
    rte_memcpy(room, &tx->ethhdr, sizeof(tx->ethhdr));
  }
  return rte_eth_tx_burst(face->port, queue, pkts, nPkts);
}
