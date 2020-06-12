#ifndef NDN_DPDK_APP_FETCH_TCPCUBIC_H
#define NDN_DPDK_APP_FETCH_TCPCUBIC_H

/// \file

#include "../../dpdk/eal/tsc.h"

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

/** \brief Window increase.
 */
void
TcpCubic_Increase(TcpCubic* ca, TscTime now, double sRtt);

/** \brief Window decrease.
 *
 *  Caller must ensure this is invoked no more than once per RTT.
 */
void
TcpCubic_Decrease(TcpCubic* ca, TscTime now);

#endif // NDN_DPDK_APP_FETCH_TCPCUBIC_H
