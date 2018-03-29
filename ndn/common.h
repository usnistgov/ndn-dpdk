#ifndef NDN_DPDK_NDN_COMMON_H
#define NDN_DPDK_NDN_COMMON_H

/// \file

#include "../core/common.h"
#include <rte_byteorder.h>

#include "../dpdk/mbuf-loc.h"

#include "error.h"

typedef struct Packet Packet;

#define RETURN_IF_ERROR                                                        \
  do {                                                                         \
    if (unlikely(e != NdnError_OK))                                            \
      return e;                                                                \
  } while (false)

#define RETURN_IF_NULL(ptr, ret)                                               \
  do {                                                                         \
    if (unlikely(ptr == NULL))                                                 \
      return ret;                                                              \
  } while (false)

#endif // NDN_DPDK_NDN_COMMON_H
