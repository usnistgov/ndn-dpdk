#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_UP_H

/// \file

#include "common.h"

/** \brief A PIT upstream record.
 */
typedef struct PitUp
{
  FaceId face;
} __rte_aligned(32) PitUp;
static_assert(sizeof(PitUp) <= 32, "");

static void
PitUp_Copy(PitUp* dst, PitUp* src)
{
  rte_mov32((uint8_t*)dst, (const uint8_t*)src);
  src->face = FACEID_INVALID;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
