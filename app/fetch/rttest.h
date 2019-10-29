#ifndef NDN_DPDK_APP_FETCH_RTTEST_H
#define NDN_DPDK_APP_FETCH_RTTEST_H

/// \file

#include "../../dpdk/tsc.h"

#define RTTEST_K 4.0
#define RTTEST_ALPHA (1.0 / 8.0)
#define RTTEST_BETA (1.0 / 4.0)
#define RTTEST_INITRTO_MS 1000
#define RTTEST_MINRTO_MS 200
#define RTTEST_MAXRTO_MS 60000
extern TscDuration RTTEST_MINRTO;
extern TscDuration RTTEST_MAXRTO;

/** \brief RTT estimator.
 *  \sa https://tools.ietf.org/html/rfc6298
 */
typedef struct RttEst
{
  TscDuration rtt; // equals sRtt
  double sRtt;
  double rttVar;
  TscDuration rto;
  TscTime next_; // when to take next RTT sample
} RttEst;

void
RttEst_Init(RttEst* rtte);

/** \brief Add RTT sample.
 *  \param since packet transmission time.
 *  \param now acknowledgement receipt time.
 *  \pre packet has not been retransmitted.
 */
inline void
RttEst_Push(RttEst* rtte, TscTime since, TscTime now)
{
  if (likely(rtte->next_ > now)) {
    return;
  }
  TscDuration rtt = now - since;

  if (unlikely(rtte->rtt == 0)) {
    rtte->sRtt = rtt;
    rtte->rttVar = rtt / 2.0;
  } else {
    rtte->rttVar =
      (1.0 - RTTEST_BETA) * rtte->rttVar + RTTEST_BETA * fabs(rtte->sRtt - rtt);
    rtte->sRtt = (1.0 - RTTEST_ALPHA) * rtte->sRtt + RTTEST_ALPHA * rtt;
  }
  rtte->rtt = rtte->sRtt;
  TscDuration rto = rtte->sRtt + RTTEST_K * rtte->rttVar;
  rtte->rto = RTE_MAX(RTTEST_MINRTO, RTE_MIN(rto, RTTEST_MAXRTO));

  rtte->next_ = now + rtt;
}

/** \brief Back off the RTO timer.
 */
inline void
RttEst_Backoff(RttEst* rtte)
{
  rtte->rto = RTE_MAX(RTTEST_MINRTO, RTE_MIN(rtte->rto * 2, RTTEST_MAXRTO));
}

#endif // NDN_DPDK_APP_FETCH_RTTEST_H
