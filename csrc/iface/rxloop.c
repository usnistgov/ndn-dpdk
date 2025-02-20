#include "rxloop.h"
#include "face-impl.h"

typedef struct RxLoopTransferCtx {
  FaceID pendingFace; ///< pkts[:nPending] shall have this faceID
  uint16_t nPending;  ///< pkts[:nPending] are to be dispatched
  uint16_t nFree;     ///< frees[:nFree] are to be freed
  RTE_MARKER zeroizeEnd_;
  struct rte_mbuf* pkts[MaxBurstSize];
  struct rte_mbuf* frees[MaxBurstSize];
} RxLoopTransferCtx;

/** @brief Dispatch a burst of packets that belong to the same face. */
__attribute__((nonnull)) static inline void
RxLoop_Dispatch(RxLoop* rxl, RxGroup* rxg, RxLoopTransferCtx* tCtx) {
  uint16_t count = tCtx->nPending;
  if (count == 0) {
    return;
  }
  tCtx->nPending = 0;

  Face* face = Face_Get(tCtx->pendingFace);
  if (unlikely(face->impl == NULL)) {
    rte_memcpy(tCtx->frees, tCtx->pkts, sizeof(tCtx->pkts[0]) * count);
    tCtx->nFree += count;
    return;
  }

  PdumpSourceRef_Process(&face->impl->rxPdump, tCtx->pkts, count);

  Packet* npkts[MaxBurstSize];
  FaceRxInputCtx fCtx = (FaceRxInputCtx){
    .pkts = tCtx->pkts,
    .npkts = npkts,
    .frees = tCtx->frees + tCtx->nFree,
    .count = count,
  };
  face->impl->rxInput(face, rxg->rxThread, &fCtx);
  NDNDPDK_ASSERT(fCtx.nL3 <= count);
  NDNDPDK_ASSERT(fCtx.nFree <= count);
  tCtx->nFree += fCtx.nFree;

  InputDemuxes* demuxes =
    likely(face->impl->rxDemuxes == NULL) ? &rxl->demuxes : face->impl->rxDemuxes;
  for (uint16_t i = 0; i < fCtx.nL3; ++i) {
    Packet* npkt = npkts[i];
    bool accepted = InputDemux_Dispatch(InputDemux_Of(demuxes, Packet_GetType(npkt)), npkt);
    if (unlikely(!accepted)) {
      tCtx->frees[tCtx->nFree++] = Packet_ToMbuf(npkt);
    }
  }
}

/** @brief Receive a burst of packets from @p rxg and dispatch them. */
__attribute__((nonnull)) static uint16_t
RxLoop_Transfer(RxLoop* rxl, RxGroup* rxg) {
  RxGroupBurstCtx bCtx;
  memset(&bCtx, 0, offsetof(RxGroupBurstCtx, zeroizeEnd_));
  rxg->rxBurst(rxg, &bCtx);

  RxLoopTransferCtx tCtx;
  memset(&tCtx, 0, offsetof(RxLoopTransferCtx, zeroizeEnd_));

  for (uint16_t i = 0; i < bCtx.nRx; ++i) {
    struct rte_mbuf* pkt = bCtx.pkts[i];

    if (unlikely(rte_bitset_test(bCtx.dropBits, i))) {
      if (likely(pkt != NULL)) {
        tCtx.frees[tCtx.nFree++] = pkt;
      } else {
        // pkt was passed to pdump
        // pkt was passed to EthPassthru
        // pkt was freed as bounceBufs in EthRxTable_RxBurst
      }
      continue;
    }

    if (unlikely(pkt->port != tCtx.pendingFace)) {
      RxLoop_Dispatch(rxl, rxg, &tCtx);
      tCtx.pendingFace = pkt->port;
    }

    tCtx.pkts[tCtx.nPending++] = pkt;
  }
  RxLoop_Dispatch(rxl, rxg, &tCtx);

  if (unlikely(tCtx.nFree > 0)) {
    rte_pktmbuf_free_bulk(tCtx.frees, tCtx.nFree);
  }
  return bCtx.nRx;
}

int
RxLoop_Run(RxLoop* rxl) {
  rcu_register_thread();
  uint16_t nProcessed = 0;
  while (ThreadCtrl_Continue(rxl->ctrl, nProcessed)) {
    rcu_quiescent_state();
    rcu_read_lock();
    RxGroup* rxg;
    struct cds_hlist_node* pos;
    cds_hlist_for_each_entry_rcu (rxg, pos, &rxl->head, rxlNode) {
      nProcessed += RxLoop_Transfer(rxl, rxg);
    }
    rcu_read_unlock();
  }
  rcu_unregister_thread();
  return 0;
}
