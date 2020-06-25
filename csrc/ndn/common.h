#ifndef NDN_DPDK_NDN_COMMON_H
#define NDN_DPDK_NDN_COMMON_H

/// \file

#include "../core/common.h"
#include <rte_byteorder.h>

#include "../dpdk/cryptodev.h"
#include "../dpdk/mbuf-loc.h"

#include "enum.h"

typedef struct Packet Packet;
typedef struct PInterest PInterest;
typedef struct PData PData;

#define RETURN_IF_ERROR                                                                            \
  do {                                                                                             \
    if (unlikely(e != NdnErrOK))                                                                   \
      return e;                                                                                    \
  } while (false)

#define RETURN_IF_NULL(ptr, ret)                                                                   \
  do {                                                                                             \
    if (unlikely(ptr == NULL))                                                                     \
      return ret;                                                                                  \
  } while (false)

#endif // NDN_DPDK_NDN_COMMON_H
