#include "rttest.h"

static_assert(RttEstMinRto > 0, "");
static_assert(RttEstInitRto > RttEstMinRto, "");
static_assert(RttEstMaxRto > RttEstInitRto, "");

TscDuration RttEstTscInitRto = 0;
TscDuration RttEstTscMinRto = 0;
TscDuration RttEstTscMaxRto = 0;

static void
RttEst_InitOnce()
{
  RttEstTscInitRto = TscDuration_FromMillis(RttEstInitRto);
  RttEstTscMinRto = TscDuration_FromMillis(RttEstMinRto);
  RttEstTscMaxRto = TscDuration_FromMillis(RttEstMaxRto);
}

void
RttEst_Init(RttEst* rtte)
{
  if (unlikely(RttEstTscMaxRto == 0)) {
    RttEst_InitOnce();
  }

  *rtte = (const RttEst){
    .rto = RttEstTscInitRto,
  };
}
