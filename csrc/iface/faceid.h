#ifndef NDN_DPDK_IFACE_FACEID_H
#define NDN_DPDK_IFACE_FACEID_H

/// \file

#include "common.h"
#include "enum.h"

/** \brief Numeric face identifier.
 *
 *  This may appear in rte_mbuf.port field.
 */
typedef uint16_t FaceID;

#define PRI_FaceID PRIu16

#endif // NDN_DPDK_IFACE_FACEID_H
