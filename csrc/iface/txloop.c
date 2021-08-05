#include "txloop.h"
#include "../hrlog/entry.h"

__attribute__((nonnull)) static void
TxLoop_TxFrames(Face* face, struct rte_mbuf** frames, uint16_t count)
{
  NDNDPDK_ASSERT(count > 0);
  TxProc* tx = &face->impl->tx;
  PdumpFaceRef_Process(&tx->pdump, face->id, frames, count);

  tx->nFrames[PktFragment] += count;
  for (uint16_t i = 0; i < count; ++i) {
    tx->nOctets += frames[i]->pkt_len;
  }

  uint16_t nQueued = (*tx->l2Burst)(face, frames, count);
  uint16_t nRejects = count - nQueued;
  if (unlikely(nRejects > 0)) {
    tx->nDroppedFrames += nRejects;
    uint32_t nDroppedOctets = 0;
    for (uint16_t i = nQueued; i < count; ++i) {
      nDroppedOctets += frames[i]->pkt_len;
    }
    tx->nDroppedOctets += nDroppedOctets;
    rte_pktmbuf_free_bulk(&frames[nQueued], nRejects);
  }
}

__attribute__((nonnull)) static uint64_t
TxLoop_Transfer(Face* face)
{
  TxProc* tx = &face->impl->tx;
  Packet* npkts[MaxBurstSize];
  uint16_t count = rte_ring_dequeue_burst(face->outputQueue, (void**)npkts, MaxBurstSize, NULL);

  struct rte_mbuf* frames[MaxBurstSize + LpMaxFragments];
  uint16_t nFrames = 0;
  HrlogEntry hrl[MaxBurstSize];
  uint16_t nHrls = 0;

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < count; ++i) {
    Packet* npkt = npkts[i];
    PktType framePktType = PktType_ToFull(Packet_GetType(npkt));
    ++tx->nFrames[framePktType];

    if (Hrlog_Enabled()) {
      struct rte_mbuf* m = Packet_ToMbuf(npkt);
      TscDuration latency = now - Mbuf_GetTimestamp(m);
      switch (framePktType) {
        case PktInterest:
          hrl[nHrls++] = HrlogEntry_New(HRLOG_OI, latency);
          break;
        case PktData:
          hrl[nHrls++] =
            HrlogEntry_New(m->port == MBUF_INVALID_PORT ? HRLOG_OC : HRLOG_OD, latency);
          break;
        case PktNack:
          break;
        default:
          NDNDPDK_ASSERT(false);
      }
    }

    nFrames += TxProc_Output(tx, npkt, &frames[nFrames], face->txAlign);
    if (unlikely(nFrames >= MaxBurstSize)) {
      TxLoop_TxFrames(face, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    TxLoop_TxFrames(face, frames, nFrames);
  }
  Hrlog_Post(hrl, nHrls);

  return count;
}

int
TxLoop_Run(TxLoop* txl)
{
  rcu_register_thread();
  while (ThreadStopFlag_ShouldContinue(&txl->stop)) {
    rcu_quiescent_state();
    uint64_t nProcessed = 0;

    rcu_read_lock();
    Face* face;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (face, pos, &txl->head, txlNode) {
      nProcessed += TxLoop_Transfer(face);
    }
    rcu_read_unlock();

    ThreadLoadStat_Report(&txl->loadStat, nProcessed);
  }
  rcu_unregister_thread();
  return 0;
}
