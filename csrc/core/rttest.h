#ifndef NDNDPDK_CORE_RTTEST_H
#define NDNDPDK_CORE_RTTEST_H

/** @file */

#include "../dpdk/tsc.h"
#include "rttest-enum.h"

#define RttEstAlpha (1.0 * RttEstAlphaDividend / RttEstAlphaDivisor)
#define RttEstBeta (1.0 * RttEstBetaDividend / RttEstBetaDivisor)

extern TscDuration RttEstTscInitRto;
extern TscDuration RttEstTscMinRto;
extern TscDuration RttEstTscMaxRto;

/**
 * @brief SRTT and RTTVAR values in RTT estimator.
 * @sa https://tools.ietf.org/html/rfc6298
 */
typedef struct RttValue
{
  float sRtt;
  float rttVar;
} RttValue;
static_assert(sizeof(RttValue) == sizeof(uint64_t), "");

/**
 * @brief Add RTT sample.
 *
 * This should be called once per RTT, using an RTT measurement from a non-retransmitted packet.
 *
 * If @p rttv is the zero value, @p rtt is assumed to be the first RTT measurement.
 * Otherwise, @p rtt is assumed to be a subsequent RTT measurement.
 */
__attribute__((nonnull)) static inline void
RttValue_Push(RttValue* rttv, TscDuration rtt)
{
  if (unlikely(*(uint64_t*)rttv == 0)) {
    rttv->sRtt = rtt;
    rttv->rttVar = rtt / 2.0;
  } else {
    rttv->rttVar = (1.0 - RttEstBeta) * rttv->rttVar + RttEstBeta * fabsf(rttv->sRtt - rtt);
    rttv->sRtt = (1.0 - RttEstAlpha) * rttv->sRtt + RttEstAlpha * rtt;
  }
}

/**
 * @brief RTT estimator.
 * @sa https://tools.ietf.org/html/rfc6298
 */
typedef struct RttEst
{
  RttValue rttv;
  TscDuration rto;
  TscDuration last; // last input RTT (for external sampling only)
  TscTime next_;    // when to take next RTT sample
} RttEst;

__attribute__((nonnull)) void
RttEst_Init(RttEst* rtte);

__attribute__((nonnull)) static inline void
RttEst_SetRTO_(RttEst* rtte, TscDuration rto)
{
  rtte->rto = RTE_MAX(RttEstTscMinRto, RTE_MIN(rto, RttEstTscMaxRto));
}

/**
 * @brief Add RTT sample.
 * @pre packet has not been retransmitted.
 */
__attribute__((nonnull)) static inline void
RttEst_Push(RttEst* rtte, TscTime now, TscDuration rtt)
{
  rtte->last = rtt;
  if (likely(rtte->next_ > now)) {
    return;
  }

  RttValue_Push(&rtte->rttv, rtt);
  RttEst_SetRTO_(rtte, rtte->rttv.sRtt + RttEstK * rtte->rttv.rttVar);
  rtte->next_ = now + rtte->rttv.sRtt;
}

/** @brief Back off the RTO timer. */
__attribute__((nonnull)) static inline void
RttEst_Backoff(RttEst* rtte)
{
  RttEst_SetRTO_(rtte, rtte->rto * 2);
}

#endif // NDNDPDK_CORE_RTTEST_H
