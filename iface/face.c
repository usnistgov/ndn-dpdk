#include "face.h"

#include "rx-proc.h"

uint16_t
Face_RxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  uint16_t nInputs = (*face->rxBurstOp)(face, pkts, nPkts);
  uint16_t nProcessed = 0;
  for (uint16_t i = 0; i < nInputs; ++i) {
    struct rte_mbuf* processed = RxProc_Input(&face->rx, pkts[i]);
    if (processed != NULL) {
      pkts[nProcessed++] = processed;
    }
  }
  return nProcessed;
}

static const int TX_BURST_FRAMES = 64;  // number of frames in a burst
static const int TX_MAX_FRAGMENTS = 64; // max allowed number of fragments

static inline void
Face_TxBurst_SendFrames(Face* face, struct rte_mbuf** frames, uint16_t nFrames)
{
  assert(nFrames > 0);
  uint16_t nQueued = (*face->txBurstOp)(face, frames, nFrames);
  uint16_t nRejects = nFrames - nQueued;
  FreeMbufs(frames + nQueued, nRejects);
  TxProc_CountQueued(&face->tx, nQueued, nRejects);
}

void
Face_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  struct rte_mbuf* frames[TX_BURST_FRAMES + TX_MAX_FRAGMENTS];
  uint16_t nFrames = 0;

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    nFrames +=
      TxProc_Output(&face->tx, pkt, frames + nFrames, TX_MAX_FRAGMENTS);

    if (unlikely(nFrames >= TX_BURST_FRAMES)) {
      Face_TxBurst_SendFrames(face, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    Face_TxBurst_SendFrames(face, frames, nFrames);
  }
}

void
Face_ReadCounters(Face* face, FaceCounters* cnt)
{
  RxProc_ReadCounters(&face->rx, cnt);
  TxProc_ReadCounters(&face->tx, cnt);
}

void
FaceImpl_Init(Face* face, uint16_t mtu, uint16_t headroom,
              struct rte_mempool* indirectMp, struct rte_mempool* headerMp)
{
  TxProc_Init(&face->tx, mtu, headroom, indirectMp, headerMp);
}
