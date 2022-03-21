#ifndef NDNDPDK_FETCH_TCPCUBIC_H
#define NDNDPDK_FETCH_TCPCUBIC_H

/** @file */

#include "../dpdk/tsc.h"

/**
 * @brief TCP CUBIC algorithm.
 * @sa https://tools.ietf.org/html/rfc8312
 */
typedef struct TcpCubic
{
  TscTime t0;
  double cwnd;
  double wMax;
  double k;
  double ssthresh;
} TcpCubic;

__attribute__((nonnull)) void
TcpCubic_Init(TcpCubic* ca);

__attribute__((nonnull)) static inline uint32_t
TcpCubic_GetCwnd(TcpCubic* ca)
{
  return RTE_MAX((uint32_t)ca->cwnd, 1);
}

/** @brief Window increase. */
__attribute__((nonnull)) void
TcpCubic_Increase(TcpCubic* ca, TscTime now, double sRtt);

/**
 * @brief Window decrease.
 *
 * Caller must ensure this is invoked no more than once per RTT.
 */
__attribute__((nonnull)) void
TcpCubic_Decrease(TcpCubic* ca, TscTime now);

#endif // NDNDPDK_FETCH_TCPCUBIC_H
