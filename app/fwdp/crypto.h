#ifndef NDN_DPDK_APP_FWDP_CRYPTO_H
#define NDN_DPDK_APP_FWDP_CRYPTO_H

/// \file

#include "input.h"

/** \brief Forwarder data plane, crypto helper.
 */
typedef struct FwCrypto
{
  struct rte_ring* input;
  struct rte_mempool* opPool;
  FwInput* output;

  bool stop;

  uint8_t devId;
  uint16_t qpId;
} FwCrypto;

void FwCrypto_Run(FwCrypto* fwc);

#endif // NDN_DPDK_APP_FWDP_CRYPTO_H
