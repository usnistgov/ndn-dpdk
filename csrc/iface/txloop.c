#include "txloop.h"
#include "../hrlog/entry.h"
#include "face-impl.h"

__attribute__((nonnull)) static void
TxLoop_TxFrames(Face* face, int txThread, struct rte_mbuf** frames, uint16_t count)
{
  NDNDPDK_ASSERT(count > 0);
  PdumpSourceRef_Process(&face->impl->txPdump, frames, count);

  FaceTxThread* txt = &face->impl->tx[txThread];
  txt->nFrames[PktFragment] += count;
  for (uint16_t i = 0; i < count; ++i) {
    txt->nOctets += frames[i]->pkt_len;
  }

  uint16_t nQueued = face->impl->txBurst(face, frames, count);
  uint16_t nRejects = count - nQueued;
  if (unlikely(nRejects > 0)) {
    txt->nDroppedFrames += nRejects;
    uint32_t nDroppedOctets = 0;
    for (uint16_t i = nQueued; i < count; ++i) {
      nDroppedOctets += frames[i]->pkt_len;
    }
    txt->nDroppedOctets += nDroppedOctets;
    rte_pktmbuf_free_bulk(&frames[nQueued], nRejects);
  }
}

__attribute__((nonnull)) static uint16_t
TxLoop_Transfer(Face* face, int txThread)
{
  FaceTxThread* txt = &face->impl->tx[txThread];
  Packet* npkts[MaxBurstSize];
  uint16_t count = rte_ring_dequeue_burst(face->outputQueue, (void**)npkts, MaxBurstSize, NULL);

  struct rte_mbuf* frames[MaxBurstSize + LpMaxFragments];
  uint16_t nFrames = 0;
  struct rte_ring* hrlRing = HrlogRing_Get();
  HrlogEntry hrl[MaxBurstSize];
  uint16_t nHrls = 0;

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < count; ++i) {
    Packet* npkt = npkts[i];
    PktType framePktType = PktType_ToFull(Packet_GetType(npkt));
    ++txt->nFrames[framePktType];

    if (hrlRing != NULL) {
      struct rte_mbuf* m = Packet_ToMbuf(npkt);
      TscDuration latency = now - Mbuf_GetTimestamp(m);
      switch (framePktType) {
        case PktInterest:
          hrl[nHrls++] = HrlogEntry_New(HRLOG_OI, latency);
          break;
        case PktData:
          hrl[nHrls++] =
            HrlogEntry_New(m->port == RTE_MBUF_PORT_INVALID ? HRLOG_OC : HRLOG_OD, latency);
          break;
        case PktNack:
          break;
        default:
          NDNDPDK_ASSERT(false);
      }
    }

    nFrames += FaceTx_Output(face, txThread, npkt, &frames[nFrames]);
    if (unlikely(nFrames >= MaxBurstSize)) {
      TxLoop_TxFrames(face, txThread, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    TxLoop_TxFrames(face, txThread, frames, nFrames);
  }
  if (hrlRing != NULL) {
    HrlogRing_Post(hrlRing, hrl, nHrls);
  }

  return count;
}

int
TxLoop_Run(TxLoop* txl)
{
  rcu_register_thread();
  uint16_t nProcessed = 0;
  while (ThreadCtrl_Continue(txl->ctrl, nProcessed)) {
    rcu_quiescent_state();
    rcu_read_lock();
    Face* face;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (face, pos, &txl->head, txlNode) {
      static_assert(MaxFaceTxThreads == 1, "");
      nProcessed += TxLoop_Transfer(face, 0);
    }
    rcu_read_unlock();
  }
  rcu_unregister_thread();
  return 0;
}
