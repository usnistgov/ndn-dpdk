#include "pktcopy-rx.h"

#define PKTCOPYRX_BURST_SIZE 64

void
PktcopyRx_AddTxRing(PktcopyRx* pcrx, struct rte_ring* r)
{
  assert(pcrx->nTxRings < PKTCOPYRX_MAXTX);
  pcrx->txRings[pcrx->nTxRings++] = r;
}

void
PktcopyRx_Run(PktcopyRx* pcrx)
{
  struct rte_mbuf* pkts[PKTCOPYRX_BURST_SIZE];
  struct rte_mbuf* indirects[PKTCOPYRX_BURST_SIZE];
  while (true) {
    uint16_t nRx =
      Face_RxBurst(pcrx->face, (Packet**)pkts, PKTCOPYRX_BURST_SIZE);
    if (nRx == 0) {
      continue;
    }
    if (pcrx->nTxRings == 0) {
      FreeMbufs(pkts, nRx);
    }

    for (int i = 0; i < pcrx->nTxRings; ++i) {
      struct rte_mbuf** txPkts = pkts;
      if (i < pcrx->nTxRings - 1) {
        int res = rte_pktmbuf_alloc_bulk(pcrx->mpIndirect, indirects, nRx);
        if (unlikely(res != 0)) {
          // TODO memory allocation error counter
          continue;
        }
        // XXX missing: make indirect mbufs point to packets
        txPkts = indirects;
      }
      unsigned nEnq =
        rte_ring_enqueue_burst(pcrx->txRings[i], (void**)txPkts, nRx, NULL);
      if (unlikely(nEnq < nRx)) {
        // TODO ring congestion counter
        FreeMbufs(txPkts + nEnq, nRx - nEnq);
      }
    }
  }
}
