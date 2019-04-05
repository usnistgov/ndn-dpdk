#ifndef NDN_DPDK_IFACE_FACEID_H
#define NDN_DPDK_IFACE_FACEID_H

/// \file

#include "common.h"

/** \brief Numeric face identifier.
 *
 *  This may appear in rte_mbuf.port field.
 */
typedef uint16_t FaceId;

#define FACEID_INVALID 0
#define FACEID_MIN 1
#define FACEID_MAX (UINT16_MAX - 1)
#define PRI_FaceId PRIu16

/** \brief Face state.
 */
typedef enum FaceState
{
  FACESTA_UNUSED = 0,
  FACESTA_UP = 1,
  FACESTA_DOWN = 2,
  FACESTA_REMOVED = 3,
} __rte_packed FaceState;
static_assert(sizeof(FaceState) == 1, "");

#endif // NDN_DPDK_IFACE_FACEID_H
