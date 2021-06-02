#ifndef NDNDPDK_FWDP_CRYPTO_H
#define NDNDPDK_FWDP_CRYPTO_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/input-demux.h"

/** @brief Forwarder data plane, crypto helper. */
typedef struct FwCrypto
{
  struct rte_ring* input;
  struct rte_mempool* opPool;
  InputDemux output;

  uint64_t nDrops;

  CryptoQueuePair singleSeg; ///< CryptoDev for single-segment packets
  CryptoQueuePair multiSeg;  ///< CryptoDev for multi-segment packets
  ThreadStopFlag stop;
  ThreadLoadStat loadStat;
} FwCrypto;

void
FwCrypto_Run(FwCrypto* fwc);

#endif // NDNDPDK_FWDP_CRYPTO_H
