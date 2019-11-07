#ifndef NDN_DPDK_APP_FETCH_TCPCUBIC_H
#define NDN_DPDK_APP_FETCH_TCPCUBIC_H

/// \file

#include "../../dpdk/tsc.h"

/** \brief TCP CUBIC algorithm.
 *  \sa https://tools.ietf.org/html/rfc8312
 */
typedef struct TcpCubic
{
  TscDuration t0;
  double cwnd;
  double wMax;
  double k;
  double ssthresh;
} TcpCubic;

void
TcpCubic_Init(TcpCubic* ca);

static inline uint32_t
TcpCubic_GetCwnd(TcpCubic* ca)
{
  return RTE_MAX((uint32_t)ca->cwnd, 1);
}

void
TcpCubic_Increase(TcpCubic* ca, TscTime now, double sRtt);

void
TcpCubic_Decrease(TcpCubic* ca, TscTime now, double sRtt);

#endif // NDN_DPDK_APP_FETCH_TCPCUBIC_H
