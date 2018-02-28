#ifndef NDN_DPDK_IFACE_COMMON_H
#define NDN_DPDK_IFACE_COMMON_H

/// \file

#include "../ndn/packet.h"
#include "../ndn/protonum.h"

/** \brief Numeric face identifier.
 */
typedef uint16_t FaceId;

#define FACEID_INVALID 0
#define FACEID_MAX UINT16_MAX
#define PRI_FaceId PRIu16

typedef struct Face Face;

#endif // NDN_DPDK_IFACE_COMMON_H
