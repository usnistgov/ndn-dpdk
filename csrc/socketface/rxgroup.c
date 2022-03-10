#include "rxgroup.h"

void
SocketRxGroup_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  SocketRxGroup* srxg = container_of(rxg, SocketRxGroup, base);
  ctx->nRx = rte_ring_dequeue_burst(srxg->ring, (void**)ctx->pkts, RTE_DIM(ctx->pkts), NULL);
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, SocketRxGroup_RxBurst);
