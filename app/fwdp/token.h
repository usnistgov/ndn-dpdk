#ifndef NDN_DPDK_APP_FWDP_TOKEN_H
#define NDN_DPDK_APP_FWDP_TOKEN_H

/// \file

#include "../../core/common.h"

static inline uint64_t
FwToken_New(uint8_t fwdId, uint64_t pccToken)
{
  return ((uint64_t)fwdId << 56) | (pccToken & 0xFFFFFFFFFFFF);
}

#endif // NDN_DPDK_APP_FWDP_TOKEN_H
