#include "rxgroup.h"

uint16_t
SocketRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  SocketRxGroup* srxg = container_of(rxg, SocketRxGroup, base);
  return rte_ring_dequeue_burst(srxg->ring, (void**)pkts, nPkts, NULL);
}
