#ifndef NDN_DPDK_APP_FWDP_CRYPTO_H
#define NDN_DPDK_APP_FWDP_CRYPTO_H

/// \file

#include "../../app/inputdemux/demux.h"
#include "../../dpdk/eal/thread.h"

/** \brief Forwarder data plane, crypto helper.
 */
typedef struct FwCrypto
{
  struct rte_ring* input;
  struct rte_mempool* opPool;
  InputDemux output;

  uint64_t nDrops;

  CryptoQueuePair singleSeg; ///< CryptoDev for single-segment packets
  CryptoQueuePair multiSeg;  ///< CryptoDev for multi-segment packets
  ThreadStopFlag stop;
} FwCrypto;

void
FwCrypto_Run(FwCrypto* fwc);

#endif // NDN_DPDK_APP_FWDP_CRYPTO_H
