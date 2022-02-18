#include "rxgroup.h"

static const RxGroup_RxBurstFunc _ __rte_unused = SocketRxGroup_RxBurst;

void
SocketRxGroup_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  SocketRxGroup* srxg = container_of(rxg, SocketRxGroup, base);
  ctx->nRx = rte_ring_dequeue_burst(srxg->ring, (void**)ctx->pkts, RTE_DIM(ctx->pkts), NULL);
}
