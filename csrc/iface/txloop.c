#include "txloop.h"
#include "../hrlog/post.h"

static const int TX_BURST_FRAMES = 64;  // number of frames in a burst
static const int TX_MAX_FRAGMENTS = 16; // max allowed number of fragments

static void
TxLoop_TxFrames(Face* face, struct rte_mbuf** frames, uint16_t count)
{
  assert(count > 0);
  TxProc* tx = &face->impl->tx;

  tx->nFrames += count;
  for (uint16_t i = 0; i < count; ++i) {
    tx->nOctets += frames[i]->pkt_len;
  }

  uint16_t nQueued = (*face->txBurstOp)(face, frames, count);
  uint16_t nRejects = count - nQueued;
  if (unlikely(nRejects > 0)) {
    tx->nDroppedFrames += nRejects;
    tx->nDroppedOctets += FreeMbufs(&frames[nQueued], nRejects);
  }
}

static void
TxLoop_Transfer(Face* face)
{
  TxProc* tx = &face->impl->tx;
  Packet* npkts[TX_BURST_FRAMES];
  uint16_t count =
    rte_ring_dequeue_burst(face->txQueue, (void**)npkts, TX_BURST_FRAMES, NULL);

  struct rte_mbuf* frames[TX_BURST_FRAMES + TX_MAX_FRAGMENTS];
  uint16_t nFrames = 0;
  HrlogEntry hrl[TX_BURST_FRAMES];
  uint16_t nHrls = 0;

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < count; ++i) {
    Packet* npkt = npkts[i];
    TscDuration latency = now - Packet_ToMbuf(npkt)->timestamp;
    L3PktType l3type = Packet_GetL3PktType(npkt);
    RunningStat_Push1(&tx->latency[l3type], latency);
    if (l3type == L3PktType_Interest) {
      hrl[nHrls++] = HrlogEntry_New(HRLOG_OI, latency);
    } else if (l3type == L3PktType_Data) {
      hrl[nHrls++] = HrlogEntry_New(
        Packet_ToMbuf(npkt)->port == MBUF_INVALID_PORT ? HRLOG_OC : HRLOG_OD,
        latency);
    }

    struct rte_mbuf** outFrames = &frames[nFrames];
    nFrames += TxProc_Output(tx, npkt, outFrames, TX_MAX_FRAGMENTS);

    if (unlikely(nFrames >= TX_BURST_FRAMES)) {
      TxLoop_TxFrames(face, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    TxLoop_TxFrames(face, frames, nFrames);
  }
  if (likely(nHrls > 0)) {
    Hrlog_PostBulk(hrl, nHrls);
  }
}

void
TxLoop_Run(TxLoop* txl)
{
  while (ThreadStopFlag_ShouldContinue(&txl->stop)) {
    rcu_quiescent_state();
    rcu_read_lock();

    Face* face;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu(face, pos, &txl->head, txlNode)
    {
      TxLoop_Transfer(face);
    }
    rcu_read_unlock();
  }
}
