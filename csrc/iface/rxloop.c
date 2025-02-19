#include "rxloop.h"
#include "face-impl.h"

__attribute__((nonnull)) static uint16_t
RxLoop_Transfer(RxLoop* rxl, RxGroup* rxg) {
  RxGroupBurstCtx ctx;
  memset(&ctx, 0, offsetof(RxGroupBurstCtx, zeroizeEnd_));
  rxg->rxBurst(rxg, &ctx);

  struct rte_mbuf* frees[MaxBurstSize];
  uint16_t nFrees = 0;
  for (uint16_t i = 0; i < ctx.nRx; ++i) {
    struct rte_mbuf* pkt = ctx.pkts[i];

    if (unlikely(rte_bitset_test(ctx.dropBits, i))) {
      if (likely(pkt != NULL)) {
        frees[nFrees++] = pkt;
      } else {
        // pkt was passed to pdump
        // pkt was passed to EthPassthru
        // pkt was freed as bounceBufs in EthRxTable_RxBurst
      }
      continue;
    }

    Face* face = Face_Get(pkt->port);
    if (unlikely(face->impl == NULL)) {
      frees[nFrees++] = pkt;
      continue;
    }

    PdumpSourceRef_Process(&face->impl->rxPdump, &pkt, 1);

    Packet* npkt = face->impl->rxInput(face, rxg->rxThread, pkt);
    NULLize(pkt);
    if (npkt == NULL) {
      continue;
    }

    InputDemuxes* demuxes =
      likely(face->impl->rxDemuxes == NULL) ? &rxl->demuxes : face->impl->rxDemuxes;
    bool accepted = InputDemux_Dispatch(InputDemux_Of(demuxes, Packet_GetType(npkt)), npkt);
    if (unlikely(!accepted)) {
      frees[nFrees++] = Packet_ToMbuf(npkt);
    }
  }

  if (unlikely(nFrees > 0)) {
    rte_pktmbuf_free_bulk(frees, nFrees);
  }
  return ctx.nRx;
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
