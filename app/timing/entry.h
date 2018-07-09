#ifndef NDN_DPDK_APP_TIMING_ENTRY_H
#define NDN_DPDK_APP_TIMING_ENTRY_H

/// \file

#include "../../dpdk/tsc.h"

/** \brief FWDP action in latency timing.
 */
typedef enum TimingAction {
  TIMING_IN = 4, // FwInput dispatch by name
  TIMING_IT = 5, // FwInput dispatch by token

  TIMING_FI = 1, // FwFwd process Interest
  TIMING_FD = 2, // FwFwd process Data
  TIMING_FN = 3, // FwFwd process Nack

  TIMING_PIT = 16, // PIT entry count
} TimingAction;

/** \brief A latency timing entry.
 */
typedef struct TimingEntry
{
  uint8_t act;   // TimingAction
  uint8_t lcore; // lcore id
  uint64_t value : 48;
} TimingEntry;
static_assert(sizeof(TimingEntry) == sizeof(void*), "");

/** \brief A latency timing file header.
 */
typedef struct TimingHeader
{
  uint32_t magic;
  uint32_t version;
  uint64_t tschz;
} TimingHeader;
static_assert(sizeof(TimingHeader) == 16, "");

#define TIMING_HEADER_MAGIC 0x35f0498a
#define TIMING_HEADER_VERSION 1

#endif // NDN_DPDK_APP_TIMING_ENTRY_H
