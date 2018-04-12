#include "pktcopy-rx.h"

static void
PktcopyRx_Transfer(PktcopyRx* pcrx, Packet** npkts, uint16_t count)
{
  int nTx = pcrx->nTxFaces + (pcrx->dumpRing != NULL);
  if (unlikely(nTx == 0)) {
    FreeMbufs((struct rte_mbuf**)npkts, count);
    return;
  }

  assert(count <= PKTCOPYRX_RXBURST_SIZE);
  Packet* clones[PKTCOPYRX_RXBURST_SIZE];
  uint16_t nOuts;
  Packet** outNpkts;

  for (int i = 0; i < nTx; ++i) {
    if (i < nTx - 1) {
      for (nOuts = 0; nOuts < count; ++nOuts) {
        clones[nOuts] =
          ClonePacket(npkts[nOuts], pcrx->headerMp, pcrx->indirectMp);
        if (unlikely(clones[nOuts] == NULL)) {
          ++pcrx->nAllocError;
          break;
        }
      }
      outNpkts = clones;
    } else {
      nOuts = count;
      outNpkts = npkts;
    }

    if (i < pcrx->nTxFaces) {
      Face_TxBurst(pcrx->txFaces[i], outNpkts, nOuts);
    } else {
      unsigned nEnq =
        rte_ring_enqueue_burst(pcrx->dumpRing, (void**)outNpkts, nOuts, NULL);
      FreeMbufs((struct rte_mbuf**)&outNpkts[nEnq], nOuts - nEnq);
    }
  }
}

void
PktcopyRx_Rx(FaceId faceId, FaceRxBurst* burst, void* pcrx0)
{
  PktcopyRx* pcrx = (PktcopyRx*)pcrx0;
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListInterests(burst), burst->nInterests);
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListData(burst), burst->nData);
  PktcopyRx_Transfer(pcrx, FaceRxBurst_ListNacks(burst), burst->nNacks);
}
