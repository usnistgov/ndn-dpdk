#ifndef NDNDPDK_IFACE_FACEID_H
#define NDNDPDK_IFACE_FACEID_H

/** @file */

#include "common.h"

/**
 * @brief Numeric face identifier.
 *
 * This may appear in rte_mbuf.port field.
 */
typedef uint16_t FaceID;

/** @brief printf format string for FaceID. */
#define PRI_FaceID PRIu16

#endif // NDNDPDK_IFACE_FACEID_H
