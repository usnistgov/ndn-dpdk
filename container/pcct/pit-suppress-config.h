#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_SUPPRESS_CONFIG_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_SUPPRESS_CONFIG_H

/// \file

#include "common.h"

/** \brief Interest suppression configuration.
 */
typedef struct PitSuppressConfig
{
  TscDuration min;   ///< initial/minimum suppression duration
  double multiplier; ///< multiplier on each transmission
  TscDuration max;   ///< maximum suppression duration
} PitSuppressConfig;

/** \brief Compute next suppression duration.
 *  \param d current suppression duration, or 0 for initial.
 */
static TscDuration
PitSuppressConfig_Compute(const PitSuppressConfig* cfg, TscDuration d)
{
  d *= cfg->multiplier;
  return RTE_MIN(cfg->max, RTE_MAX(cfg->min, d));
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_SUPPRESS_CONFIG_H
