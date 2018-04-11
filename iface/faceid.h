#ifndef NDN_DPDK_IFACE_FACEID_H
#define NDN_DPDK_IFACE_FACEID_H

/// \file

#include "common.h"

/** \brief Numeric face identifier.
 */
typedef uint16_t FaceId;

#define FACEID_INVALID 0
#define FACEID_MIN 1
#define FACEID_MAX UINT16_MAX
#define PRI_FaceId PRIu16

#endif // NDN_DPDK_IFACE_FACEID_H
