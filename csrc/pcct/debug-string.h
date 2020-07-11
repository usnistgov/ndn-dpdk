#ifndef NDN_DPDK_PCCT_DEBUG_STRING_H
#define NDN_DPDK_PCCT_DEBUG_STRING_H

/** @file */

#include "common.h"

/** @brief Capacity of per-lcore debug string. */
#define PccDebugStringLength (2 * LNAME_MAX_STRING_SIZE + 1024)

/** @brief Clear current lcore's debug string. */
void
PccDebugString_Clear();

/**
 * @brief Append text to current lcore's debug string.
 * @return beginning of current lcore's debug string
 */
const char*
PccDebugString_Appendf(const char* fmt, ...);

#endif // NDN_DPDK_PCCT_DEBUG_STRING_H
