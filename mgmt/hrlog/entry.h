#ifndef NDN_DPDK_MGMT_HRLOG_ENTRY_H
#define NDN_DPDK_MGMT_HRLOG_ENTRY_H

/// \file

#include "../../core/common.h"

/** \brief Action identifier in high resolution log.
 */
typedef enum HrlogAction
{
  HRLOG_OI = 1, // Interest TX since RX
  HRLOG_OD = 2, // retrieved Data TX since RX
  HRLOG_OC = 4, // cached Data TX since Interest RX
} HrlogAction;

/** \brief A high resolution log entry.
 */
typedef uint64_t HrlogEntry;

static inline HrlogEntry
HrlogEntry_New(HrlogAction act, uint64_t value)
{
  uint8_t lcore = rte_lcore_id();
  return (value << 16) | ((uint64_t)lcore << 8) | ((uint8_t)act);
}

/** \brief A high resolution log file header.
 */
typedef struct HrlogHeader
{
  uint32_t magic;
  uint32_t version;
  uint64_t tschz;
} HrlogHeader;
static_assert(sizeof(HrlogHeader) == 16, "");

#define HRLOG_HEADER_MAGIC 0x35f0498a
#define HRLOG_HEADER_VERSION 2

#endif // NDN_DPDK_MGMT_HRLOG_ENTRY_H
