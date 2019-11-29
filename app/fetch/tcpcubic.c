#include "tcpcubic.h"

#define TCPCUBIC_IW 2.0
#define TCPCUBIC_C 0.4
#define TCPCUBIC_BETACUBIC 0.7
static double TCPCUBIC_TSCHZ = NAN;

void
TcpCubic_Init(TcpCubic* ca)
{
  TCPCUBIC_TSCHZ = rte_get_tsc_hz();
  ca->t0 = 0;
  ca->cwnd = TCPCUBIC_IW;
  ca->wMax = NAN;
  ca->k = NAN;
  ca->ssthresh = DBL_MAX;
}

static double
TcpCubic_ComputeWCubic(TcpCubic* ca, double t)
{
  double tk = t - ca->k;
  return TCPCUBIC_C * tk * tk * tk + ca->wMax;
}

static double
TcpCubic_ComputeWEst(TcpCubic* ca, double t, double rtt)
{
  return ca->wMax * TCPCUBIC_BETACUBIC +
         (3.0 * (1.0 - TCPCUBIC_BETACUBIC) / (1.0 + TCPCUBIC_BETACUBIC)) *
           (t / rtt);
}

void
TcpCubic_Increase(TcpCubic* ca, TscTime now, double sRtt)
{
  if (ca->cwnd < ca->ssthresh) { // slow start
    ca->cwnd += 1.0;
    return;
  }
  assert(isfinite(ca->wMax));
  assert(isfinite(ca->k));

  double t = (now - ca->t0) / TCPCUBIC_TSCHZ;
  double rtt = sRtt / TCPCUBIC_TSCHZ;

  double wEst = TcpCubic_ComputeWEst(ca, t, rtt);
  if (TcpCubic_ComputeWCubic(ca, t) < wEst) { // TCP friendly region
    ca->cwnd = wEst;
    return;
  }

  // concave region or convex region
  // double wCubic = TcpCubic_ComputeWCubic(ca, t + rtt);
  double wCubic = TcpCubic_ComputeWCubic(ca, t + rtt);
  ca->cwnd += (wCubic - ca->cwnd) / ca->cwnd;
}

void
TcpCubic_Decrease(TcpCubic* ca, TscTime now)
{
  ca->t0 = now;

  ca->wMax = ca->cwnd;
  ca->k = cbrt((1 - TCPCUBIC_BETACUBIC) / TCPCUBIC_C * ca->wMax);
  ca->cwnd *= TCPCUBIC_BETACUBIC;
  ca->ssthresh = RTE_MAX(ca->cwnd, 2.0);
}
