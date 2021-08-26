#ifndef NDNDPDK_FWDP_CRYPTO_H
#define NDNDPDK_FWDP_CRYPTO_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/input-demux.h"

/** @brief Forwarder data plane, crypto helper. */
typedef struct FwCrypto
{
  ThreadCtrl ctrl;
  struct rte_ring* input;
  struct rte_mempool* opPool;
  InputDemux output;

  uint64_t nDrops;
  CryptoQueuePair cqp;
} FwCrypto;

void
FwCrypto_Run(FwCrypto* fwc);

#endif // NDNDPDK_FWDP_CRYPTO_H
