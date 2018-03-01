#include "pktcopy-rx.h"

void
PktcopyRx_AddTxRing(PktcopyRx* pcrx, struct rte_ring* r)
{
  assert(pcrx->nTxRings < PKTCOPYRX_MAXTX);
  pcrx->txRings[pcrx->nTxRings++] = r;
}

static void
PktcopyRx_Transfer(PktcopyRx* pcrx, Packet** npkts, uint16_t count)
{
  if (unlikely(pcrx->nTxRings == 0)) {
    FreeMbufs((struct rte_mbuf**)npkts, count);
    return;
  }

  for (int i = pcrx->nTxRings - 1; i > 0; --i) {
    struct rte_ring* r = pcrx->txRings[i];
    for (uint16_t j = 0; j < count; ++j) {
      struct rte_mbuf* pkt = Packet_ToMbuf(npkts[j]);
      struct rte_mbuf* clone = rte_pktmbuf_clone(pkt, pcrx->indirectMp);
      if (unlikely(clone == NULL)) {
        ++pcrx->nAllocError;
      }

      int res = rte_ring_enqueue(r, clone);
      if (unlikely(res != 0)) {
        rte_pktmbuf_free(clone);
        ++pcrx->nTxRingCongestions[i];
      }
    }
  }

  struct rte_ring* r = pcrx->txRings[0];
  unsigned nEnq = rte_ring_enqueue_burst(r, (void**)npkts, count, NULL);
  if (unlikely(nEnq < count)) {
    FreeMbufs((struct rte_mbuf**)npkts + nEnq, count - nEnq);
    ++pcrx->nTxRingCongestions[0];
  }
}

void
PktcopyRx_Rx(Face* face, FaceRxBurst* burst, void* pcrx0)
{
  PktcopyRx* pcrx = (PktcopyRx*)pcrx0;
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListInterests(burst), burst->nInterests);
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListData(burst), burst->nData);
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListNacks(burst), burst->nNacks);
}
