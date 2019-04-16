#include "txloop.h"

static const int TX_BURST_FRAMES = 64;  // number of frames in a burst
static const int TX_MAX_FRAGMENTS = 16; // max allowed number of fragments

static void
TxLoop_TxFrames(Face* face, struct rte_mbuf** frames, uint16_t nFrames)
{
  assert(nFrames > 0);
  uint16_t nQueued = (*face->txBurstOp)(face, frames, nFrames);
  uint16_t nRejects = nFrames - nQueued;
  FreeMbufs(&frames[nQueued], nRejects);
  TxProc_CountQueued(&face->impl->tx, nQueued, nRejects);
}

static void
TxLoop_Transfer(Face* face)
{
  Packet* npkts[TX_BURST_FRAMES];
  uint16_t count = rte_ring_sc_dequeue_burst(
    face->txQueue, (void**)npkts, TX_BURST_FRAMES, NULL);

  struct rte_mbuf* frames[TX_BURST_FRAMES + TX_MAX_FRAGMENTS];
  uint16_t nFrames = 0;

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < count; ++i) {
    Packet* npkt = npkts[i];
    TscDuration timeSinceRx = now - Packet_ToMbuf(npkt)->timestamp;
    RunningStat_Push1(&face->impl->latencyStat, timeSinceRx);

    struct rte_mbuf** outFrames = &frames[nFrames];
    nFrames +=
      TxProc_Output(&face->impl->tx, npkt, outFrames, TX_MAX_FRAGMENTS);

    if (unlikely(nFrames >= TX_BURST_FRAMES)) {
      TxLoop_TxFrames(face, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    TxLoop_TxFrames(face, frames, nFrames);
  }
}

void
TxLoop_Run(TxLoop* txl)
{
  while (ThreadStopFlag_ShouldContinue(&txl->stop)) {
    rcu_quiescent_state();
    rcu_read_lock();
    Face* face;
    cds_hlist_for_each_entry_rcu_2(face, &txl->head, txLoopNode)
    {
      TxLoop_Transfer(face);
    }
    rcu_read_unlock();
  }
}
