#include "pktcopy-tx.h"

#define PKTCOPYTX_BURST_SIZE 64

void
PktcopyTx_Run(PktcopyTx* pctx)
{
  struct rte_mbuf* pkts[PKTCOPYTX_BURST_SIZE];
  while (true) {
    unsigned nDeq = rte_ring_dequeue_burst(pctx->txRing, (void**)pkts,
                                           PKTCOPYTX_BURST_SIZE, NULL);
    Face_TxBurst(pctx->face, pkts, nDeq);
    FreeMbufs(pkts, nDeq);
  }
}
