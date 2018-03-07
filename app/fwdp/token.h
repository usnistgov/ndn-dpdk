#ifndef NDN_DPDK_APP_FWDP_TOKEN_H
#define NDN_DPDK_APP_FWDP_TOKEN_H

#include "../../core/common.h"

typedef union FwToken
{
  struct
  {
    uint8_t fwdId : 8;
    uint8_t _reserved : 8;
    uint64_t pccToken : 48;
  } __rte_packed;
  uint64_t token;
} FwToken;

#endif // NDN_DPDK_APP_FWDP_TOKEN_H
