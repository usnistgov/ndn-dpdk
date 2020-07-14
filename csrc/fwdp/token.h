#ifndef NDNDPDK_FWDP_TOKEN_H
#define NDNDPDK_FWDP_TOKEN_H

/** @file */

#include "../core/common.h"

static __rte_always_inline uint64_t
FwToken_New(uint8_t fwdId, uint64_t pccToken)
{
  return ((uint64_t)fwdId << 56) | (pccToken & 0xFFFFFFFFFFFF);
}

#endif // NDNDPDK_FWDP_TOKEN_H
