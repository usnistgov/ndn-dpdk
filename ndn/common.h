#ifndef NDN_DPDK_NDN_COMMON_H
#define NDN_DPDK_NDN_COMMON_H

#include "../core/common.h"
#include <rte_byteorder.h>

#include "../dpdk/mbuf-loc.h"

#include "error.h"

#define RETURN_IF_ERROR                                                        \
  do {                                                                         \
    if (e != NdnError_OK)                                                      \
      return e;                                                                \
  } while (false)
#define RETURN_IF_UNLIKELY_ERROR                                               \
  do {                                                                         \
    if (unlikely(e != NdnError_OK))                                            \
      return e;                                                                \
  } while (false)

#endif // NDN_DPDK_NDN_COMMON_H
