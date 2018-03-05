#include "pktcopy-tx.h"

#define PKTCOPYTX_BURST_SIZE 64

void
PktcopyTx_Run(PktcopyTx* pctx)
{
  Packet* npkts[PKTCOPYTX_BURST_SIZE];
  while (true) {
    unsigned nDeq = rte_ring_dequeue_burst(pctx->txRing, (void**)npkts,
                                           PKTCOPYTX_BURST_SIZE, NULL);
    Face_TxBurst(pctx->face, npkts, nDeq);
  }
}
