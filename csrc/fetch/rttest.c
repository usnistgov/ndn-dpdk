#include "rttest.h"

TscDuration RTTEST_MINRTO = 0;
TscDuration RTTEST_MAXRTO = 0;

void
RttEst_Init(RttEst* rtte)
{
  if (unlikely(RTTEST_MAXRTO == 0)) {
    RTTEST_MINRTO = TscDuration_FromMillis(RTTEST_MINRTO_MS);
    RTTEST_MAXRTO = TscDuration_FromMillis(RTTEST_MAXRTO_MS);
  }
  rtte->last = 0;
  rtte->sRtt = 1.0;
  rtte->rttVar = 0.0;
  rtte->rto = TscDuration_FromMillis(RTTEST_INITRTO_MS);
  rtte->next_ = 0;
}
