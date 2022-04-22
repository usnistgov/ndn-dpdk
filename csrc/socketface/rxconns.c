#include "rxconns.h"

void
SocketRxConns_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  SocketRxConns* rxc = container_of(rxg, SocketRxConns, base);
  ctx->nRx = rte_ring_dequeue_burst(rxc->ring, (void**)ctx->pkts, RTE_DIM(ctx->pkts), NULL);
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, SocketRxConns_RxBurst);
